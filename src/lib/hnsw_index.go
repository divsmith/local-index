package lib

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"code-search/src/models"
)

// HNSWIndex implements Hierarchical Navigable Small World algorithm for approximate nearest neighbor search
type HNSWIndex struct {
	layers        []*GraphLayer
	entryPoint    *Node
	maxLayers     int
	efConstruction int
	efSearch      int
	m             int           // max connections per layer
	maxM0         int           // max connections for layer 0
	mult          float64       // level generation factor
	layerProbability float64    // probability for level generation
	nodeCount     int64
	mu            sync.RWMutex
	rng           *rand.Rand
	vectorPool    *VectorPool
}

// GraphLayer represents a single layer in the HNSW graph
type GraphLayer struct {
	nodes map[string]*Node
	edges map[string][]string // adjacency list
	mu    sync.RWMutex
}

// Node represents a node in the HNSW graph
type Node struct {
	ID       string
	Vector   []float64
	Level    int
	Neighbors map[int][]string // neighbors by level
	Metadata map[string]interface{}
	Created  time.Time
}

// HNSWConfig contains configuration for HNSW index
type HNSWConfig struct {
	MaxLayers     int   // Maximum number of layers (default: 16)
	EFConstruction int   // Size of dynamic candidate list during construction (default: 200)
	EFSearch      int   // Size of dynamic candidate list during search (default: 50)
	M             int   // Max connections per layer (default: 16)
	MaxM0         int   // Max connections for layer 0 (default: 32)
}

// DefaultHNSWConfig returns default configuration for HNSW
func DefaultHNSWConfig() HNSWConfig {
	return HNSWConfig{
		MaxLayers:     16,
		EFConstruction: 200,
		EFSearch:      50,
		M:             16,
		MaxM0:         32,
	}
}

// NewHNSWIndex creates a new HNSW index
func NewHNSWIndex(config HNSWConfig, poolManager *PoolManager) *HNSWIndex {
	if config.MaxLayers == 0 {
		config = DefaultHNSWConfig()
	}

	// Calculate mult for level generation: P(level = l) = mult^(-l)
	mult := 1.0 / math.Log(float64(config.M))
	layerProbability := 1.0 - mult

	index := &HNSWIndex{
		layers:        make([]*GraphLayer, config.MaxLayers),
		maxLayers:     config.MaxLayers,
		efConstruction: config.EFConstruction,
		efSearch:      config.EFSearch,
		m:             config.M,
		maxM0:         config.MaxM0,
		mult:          mult,
		layerProbability: layerProbability,
		rng:           rand.New(rand.NewSource(time.Now().UnixNano())),
		vectorPool:    poolManager.GetVectorPool(),
	}

	// Initialize layers
	for i := 0; i < config.MaxLayers; i++ {
		index.layers[i] = &GraphLayer{
			nodes: make(map[string]*Node),
			edges: make(map[string][]string),
		}
	}

	return index
}

// Insert inserts a vector into the HNSW index
func (h *HNSWIndex) Insert(id string, vector []float64, metadata map[string]interface{}) error {
	if len(vector) == 0 {
		return fmt.Errorf("vector cannot be empty")
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Determine the level for this node
	level := h.getRandomLevel()

	// Create node
	node := &Node{
		ID:       id,
		Vector:   make([]float64, len(vector)),
		Level:    level,
		Neighbors: make(map[int][]string),
		Metadata: make(map[string]interface{}),
		Created:  time.Now(),
	}

	// Copy vector and metadata
	copy(node.Vector, vector)
	for k, v := range metadata {
		node.Metadata[k] = v
	}

	// Add node to appropriate layers
	for l := 0; l <= level; l++ {
		h.layers[l].mu.Lock()
		h.layers[l].nodes[id] = node
		h.layers[l].edges[id] = make([]string, 0)
		h.layers[l].mu.Unlock()
	}

	// If this is the first node, make it the entry point
	if h.entryPoint == nil {
		h.entryPoint = node
		atomic.AddInt64(&h.nodeCount, 1)
		return nil
	}

	// Find entry points for each level
	currentClosest := h.entryPoint
	for levelC := level; levelC < h.entryPoint.Level; levelC++ {
		currentClosest = h.searchLayerOne(currentClosest, vector, 1, levelC)
	}

	// For each level, insert the node
	for levelC := h.minInt(level, h.entryPoint.Level); levelC >= 0; levelC-- {
		candidates := h.searchLayer(currentClosest, vector, h.efConstruction, levelC)
		h.selectNeighbors(node, candidates, levelC)

		// Add bidirectional connections
		for _, neighborID := range node.Neighbors[levelC] {
			h.layers[levelC].mu.Lock()
			if _, exists := h.layers[levelC].edges[neighborID]; exists {
				h.layers[levelC].edges[neighborID] = append(h.layers[levelC].edges[neighborID], id)
			}
			h.layers[levelC].mu.Unlock()
		}
	}

	// Update entry point if this node is at a higher level
	if level > h.entryPoint.Level {
		h.entryPoint = node
	}

	atomic.AddInt64(&h.nodeCount, 1)
	return nil
}

// Search performs approximate nearest neighbor search
func (h *HNSWIndex) Search(queryVector []float64, limit int) ([]models.VectorSearchResult, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector cannot be empty")
	}

	if limit <= 0 {
		limit = 10
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.entryPoint == nil {
		return []models.VectorSearchResult{}, nil
	}

	// Start from entry point and go down to level 0
	currentClosest := h.entryPoint
	for level := h.entryPoint.Level; level > 0; level-- {
		currentClosest = h.searchLayerOne(currentClosest, queryVector, 1, level)
	}

	// Final search at level 0
	candidates := h.searchLayer(currentClosest, queryVector, h.efSearch, 0)

	// Convert candidates to results and sort by distance
	results := make([]models.VectorSearchResult, 0, len(candidates))
	for _, candidate := range candidates {
		if node, exists := h.layers[0].nodes[candidate.ID]; exists {
			// Convert distance to similarity (cosine similarity)
			similarity := 1.0 - candidate.Distance // Simple conversion, can be improved

			result := models.VectorSearchResult{
				ID:       candidate.ID,
				Score:    similarity,
				Metadata: node.Metadata,
			}
			results = append(results, result)
		}
	}

	// Sort by similarity (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// Candidate represents a candidate node during search
type Candidate struct {
	ID       string
	Distance float64
}

// searchLayerOne searches for the closest neighbor in a single layer
func (h *HNSWIndex) searchLayerOne(entry *Node, query []float64, ef int, level int) *Node {
	h.layers[level].mu.RLock()
	defer h.layers[level].mu.RUnlock()

	closest := entry
	closestDist := h.distance(query, closest.Vector)

	visited := make(map[string]bool)
	queue := []*Node{entry}
	visited[entry.ID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentDist := h.distance(query, current.Vector)
		if currentDist < closestDist {
			closest = current
			closestDist = currentDist
		}

		// Check neighbors
		if neighbors, exists := h.layers[level].edges[current.ID]; exists {
			for _, neighborID := range neighbors {
				if !visited[neighborID] {
					visited[neighborID] = true
					if neighbor, exists := h.layers[level].nodes[neighborID]; exists {
						neighborDist := h.distance(query, neighbor.Vector)
						if neighborDist < closestDist {
							queue = append(queue, neighbor)
						}
					}
				}
			}
		}
	}

	return closest
}

// searchLayer searches for nearest neighbors in a layer
func (h *HNSWIndex) searchLayer(entry *Node, query []float64, ef int, level int) []*Candidate {
	h.layers[level].mu.RLock()
	defer h.layers[level].mu.RUnlock()

	visited := make(map[string]bool)
	candidates := make([]*Candidate, 0, ef)
	w := make([]*Candidate, 0, ef)

	// Initialize with entry point
	entryDist := h.distance(query, entry.Vector)
	entryCandidate := &Candidate{ID: entry.ID, Distance: entryDist}
	candidates = append(candidates, entryCandidate)
	w = append(w, entryCandidate)
	visited[entry.ID] = true

	for len(candidates) > 0 {
		// Get the closest candidate
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].Distance < candidates[j].Distance
		})

		current := candidates[0]
		candidates = candidates[1:]

		// Check if we can stop
		if len(w) >= ef && current.Distance > w[len(w)-1].Distance {
			break
		}

		// Check neighbors
		if neighbors, exists := h.layers[level].edges[current.ID]; exists {
			for _, neighborID := range neighbors {
				if !visited[neighborID] {
					visited[neighborID] = true
					if neighbor, exists := h.layers[level].nodes[neighborID]; exists {
						neighborDist := h.distance(query, neighbor.Vector)

						// Add to candidates if closer than current farthest in w
						if len(w) < ef || neighborDist < w[len(w)-1].Distance {
							neighborCandidate := &Candidate{ID: neighborID, Distance: neighborDist}
							candidates = append(candidates, neighborCandidate)

							// Insert into w in sorted order
							inserted := false
							for i, existing := range w {
								if neighborDist < existing.Distance {
									w = append(w[:i], append([]*Candidate{neighborCandidate}, w[i:]...)...)
									inserted = true
									break
								}
							}
							if !inserted {
								w = append(w, neighborCandidate)
							}

							// Keep w size limited to ef
							if len(w) > ef {
								w = w[:ef]
							}
						}
					}
				}
			}
		}
	}

	return w
}

// selectNeighbors selects the best neighbors for a node at a given level
func (h *HNSWIndex) selectNeighbors(node *Node, candidates []*Candidate, level int) {
	maxNeighbors := h.m
	if level == 0 {
		maxNeighbors = h.maxM0
	}

	// Sort candidates by distance
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Distance < candidates[j].Distance
	})

	// Select up to maxNeighbors closest candidates
	selected := make([]string, 0, maxNeighbors)
	for i, candidate := range candidates {
		if i >= maxNeighbors {
			break
		}
		selected = append(selected, candidate.ID)
	}

	if node.Neighbors == nil {
		node.Neighbors = make(map[int][]string)
	}
	node.Neighbors[level] = selected
}

// getRandomLevel generates a random level for a new node
func (h *HNSWIndex) getRandomLevel() int {
	level := 0
	for h.rng.Float64() < h.layerProbability && level < h.maxLayers-1 {
		level++
	}
	return level
}

// distance calculates Euclidean distance between two vectors
func (h *HNSWIndex) distance(a, b []float64) float64 {
	if len(a) != len(b) {
		return math.MaxFloat64
	}

	var sum float64
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// cosineSimilarity calculates cosine similarity between two vectors
func (h *HNSWIndex) cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Delete removes a vector from the HNSW index
func (h *HNSWIndex) Delete(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Find and remove the node from all layers
	var nodeFound bool
	for level := 0; level < h.maxLayers; level++ {
		h.layers[level].mu.Lock()
		if _, exists := h.layers[level].nodes[id]; exists {
			nodeFound = true
			// Remove node
			delete(h.layers[level].nodes, id)

			// Remove edges to this node
			for neighborID, neighbors := range h.layers[level].edges {
				for i, neighbor := range neighbors {
					if neighbor == id {
						h.layers[level].edges[neighborID] = append(neighbors[:i], neighbors[i+1:]...)
						break
					}
				}
			}

			// Remove edges from this node
			delete(h.layers[level].edges, id)
		}
		h.layers[level].mu.Unlock()
	}

	if !nodeFound {
		return fmt.Errorf("node not found: %s", id)
	}

	// Update entry point if necessary
	if h.entryPoint != nil && h.entryPoint.ID == id {
		// Find new entry point (highest level node)
		h.entryPoint = nil
		for level := h.maxLayers - 1; level >= 0; level-- {
			h.layers[level].mu.RLock()
			for _, newNode := range h.layers[level].nodes {
				if h.entryPoint == nil || newNode.Level > h.entryPoint.Level {
					h.entryPoint = newNode
				}
			}
			h.layers[level].mu.RUnlock()
			if h.entryPoint != nil {
				break
			}
		}
	}

	atomic.AddInt64(&h.nodeCount, -1)
	return nil
}

// GetStats returns statistics about the HNSW index
func (h *HNSWIndex) GetStats() HNSWStats {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := HNSWStats{
		NodeCount:     atomic.LoadInt64(&h.nodeCount),
		MaxLayers:     h.maxLayers,
		EFConstruction: h.efConstruction,
		EFSearch:      h.efSearch,
		LayerStats:    make([]LayerStats, h.maxLayers),
	}

	for level := 0; level < h.maxLayers; level++ {
		h.layers[level].mu.RLock()
		stats.LayerStats[level].NodeCount = len(h.layers[level].nodes)
		stats.LayerStats[level].EdgeCount = 0

		for _, edges := range h.layers[level].edges {
			stats.LayerStats[level].EdgeCount += len(edges)
		}

		stats.LayerStats[level].AvgDegree = 0
		if len(h.layers[level].nodes) > 0 {
			stats.LayerStats[level].AvgDegree = float64(stats.LayerStats[level].EdgeCount) / float64(len(h.layers[level].nodes))
		}
		h.layers[level].mu.RUnlock()
	}

	if h.entryPoint != nil {
		stats.EntryLevel = h.entryPoint.Level
	}

	return stats
}

// minInt returns the minimum of two integers
func (h *HNSWIndex) minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HNSWStats contains statistics about the HNSW index
type HNSWStats struct {
	NodeCount      int64        `json:"node_count"`
	MaxLayers      int          `json:"max_layers"`
	EntryLevel     int          `json:"entry_level"`
	EFConstruction int          `json:"ef_construction"`
	EFSearch       int          `json:"ef_search"`
	LayerStats     []LayerStats `json:"layer_stats"`
}

// LayerStats contains statistics about a specific layer
type LayerStats struct {
	NodeCount  int     `json:"node_count"`
	EdgeCount  int     `json:"edge_count"`
	AvgDegree  float64 `json:"avg_degree"`
}

