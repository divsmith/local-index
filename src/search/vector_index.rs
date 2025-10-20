// ABOUTME: Optimized vector index for fast similarity search

use crate::error::Result;
use crate::models::embeddings::cosine_similarity;
use std::collections::{BTreeMap, HashMap, BinaryHeap};
use std::path::{Path, PathBuf};

#[derive(Debug, Clone)]
pub struct VectorIndex {
    dimension: usize,
    vectors: Vec<IndexedVector>,
    inverted_index: HashMap<usize, Vec<usize>>, // For efficient lookup
    max_vectors: usize,
}

#[derive(Debug, Clone)]
pub struct IndexedVector {
    pub id: usize,
    pub embedding: Vec<f32>,
    pub file_path: PathBuf,
    pub start_line: usize,
    pub end_line: usize,
    pub symbol_name: Option<String>,
    pub chunk_type: String,
}

#[derive(Debug)]
pub struct SearchResult {
    pub vector: IndexedVector,
    pub score: f32,
}

impl VectorIndex {
    pub fn new(dimension: usize, max_vectors: usize) -> Self {
        Self {
            dimension,
            vectors: Vec::with_capacity(max_vectors),
            inverted_index: HashMap::new(),
            max_vectors,
        }
    }

    pub fn add_vector(&mut self, vector: IndexedVector) -> Result<()> {
        if vector.embedding.len() != self.dimension {
            return Err(crate::error::CodeSearchError::Storage(
                format!("Vector dimension mismatch: expected {}, got {}",
                       self.dimension, vector.embedding.len())
            ));
        }

        let id = self.vectors.len();
        let mut vector = vector;
        vector.id = id;

        // Build simple inverted index based on embedding features
        self.build_inverted_index_entry(id, &vector.embedding);

        self.vectors.push(vector);
        Ok(())
    }

    fn build_inverted_index_entry(&mut self, id: usize, embedding: &[f32]) {
        // Create a simple inverted index by sampling embedding dimensions
        // This helps prune search space
        let sample_size = std::cmp::min(10, embedding.len());
        let step = embedding.len() / sample_size;

        for i in (0..embedding.len()).step_by(step) {
            let bucket = (embedding[i] * 100.0) as i64;
            let key = i * 1000 + bucket as usize;

            self.inverted_index
                .entry(key)
                .or_insert_with(Vec::new)
                .push(id);
        }
    }

    pub fn search(&self, query_embedding: &[f32], top_k: usize) -> Result<Vec<SearchResult>> {
        if query_embedding.len() != self.dimension {
            return Err(crate::error::CodeSearchError::Storage(
                format!("Query dimension mismatch: expected {}, got {}",
                       self.dimension, query_embedding.len())
            ));
        }

        // Use approximate nearest neighbor search
        let candidates = self.find_candidates(query_embedding);

        // Calculate similarities for candidates only
        let mut results = Vec::new();
        for &candidate_id in &candidates {
            let vector = &self.vectors[candidate_id];
            let similarity = cosine_similarity(query_embedding, &vector.embedding);

            if similarity > 0.1 { // Minimum similarity threshold
                results.push(SearchResult {
                    vector: vector.clone(),
                    score: similarity,
                });
            }
        }

        // Sort by similarity and take top_k
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        results.truncate(top_k);

        Ok(results)
    }

    fn find_candidates(&self, query_embedding: &[f32]) -> Vec<usize> {
        let mut candidate_counts: HashMap<usize, usize> = HashMap::new();

        // Use inverted index to find likely candidates
        let sample_size = std::cmp::min(10, query_embedding.len());
        let step = query_embedding.len() / sample_size;

        for i in (0..query_embedding.len()).step_by(step) {
            let bucket = (query_embedding[i] * 100.0) as i64;
            let key = i * 1000 + bucket as usize;

            if let Some(similar_vectors) = self.inverted_index.get(&key) {
                for &vector_id in similar_vectors {
                    *candidate_counts.entry(vector_id).or_insert(0) += 1;
                }
            }
        }

        // Sort candidates by how many buckets they appeared in
        let mut candidates: Vec<_> = candidate_counts.into_iter()
            .filter(|(_, count)| *count >= 2) // Must appear in at least 2 buckets
            .collect();

        candidates.sort_by(|a, b| b.1.cmp(&a.1));

        // Take top candidates for detailed comparison
        candidates.into_iter()
            .take(std::cmp::min(100, self.vectors.len()))
            .map(|(id, _)| id)
            .collect()
    }

    pub fn optimize_search(&mut self) -> Result<()> {
        // Sort vectors for better cache locality
        self.vectors.sort_by(|a, b| {
            // Sort by symbol name first, then by file path
            match (&a.symbol_name, &b.symbol_name) {
                (Some(a_name), Some(b_name)) => a_name.cmp(b_name),
                (Some(_), None) => std::cmp::Ordering::Less,
                (None, Some(_)) => std::cmp::Ordering::Greater,
                (None, None) => a.file_path.cmp(&b.file_path),
            }
        });

        // Rebuild inverted index with new IDs after sorting
        self.inverted_index.clear();
        let vectors_with_embeddings: Vec<(usize, Vec<f32>)> = self.vectors.iter()
            .enumerate()
            .map(|(id, v)| (id, v.embedding.clone()))
            .collect();

        for (id, embedding) in vectors_with_embeddings {
            self.build_inverted_index_entry(id, &embedding);
        }

        Ok(())
    }

    pub fn size(&self) -> usize {
        self.vectors.len()
    }

    pub fn dimension(&self) -> usize {
        self.dimension
    }

    pub fn is_empty(&self) -> bool {
        self.vectors.is_empty()
    }
}

// Approximate Nearest Neighbor (ANN) search implementation
pub struct ANNIndex {
    index: VectorIndex,
    clustering_threshold: f32,
}

impl ANNIndex {
    pub fn new(dimension: usize, max_vectors: usize) -> Self {
        Self {
            index: VectorIndex::new(dimension, max_vectors),
            clustering_threshold: 0.7,
        }
    }

    pub fn add_vector(&mut self, vector: IndexedVector) -> Result<()> {
        self.index.add_vector(vector)
    }

    pub fn search(&self, query_embedding: &[f32], top_k: usize) -> Result<Vec<SearchResult>> {
        // Multi-stage search for better performance
        let mut candidates = Vec::new();

        // Stage 1: Fast candidate selection
        let initial_results = self.index.search(query_embedding, top_k * 3)?;

        // Stage 2: Rerank with more precise scoring
        for result in initial_results {
            // Apply additional scoring heuristics
            let adjusted_score = self.adjust_score(&result, query_embedding);

            if adjusted_score > 0.2 { // Higher threshold for final results
                candidates.push(SearchResult {
                    vector: result.vector,
                    score: adjusted_score,
                });
            }
        }

        // Stage 3: Sort and limit
        candidates.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        candidates.truncate(top_k);

        Ok(candidates)
    }

    fn adjust_score(&self, result: &SearchResult, query_embedding: &[f32]) -> f32 {
        let base_score = result.score;

        // Apply various scoring adjustments
        let mut adjusted_score = base_score;

        // Boost symbol matches
        if result.vector.symbol_name.is_some() {
            adjusted_score *= 1.2;
        }

        // Boost function/class chunks
        match result.vector.chunk_type.as_str() {
            "Function" | "Class" | "Struct" => adjusted_score *= 1.1,
            _ => {}
        }

        // Apply length penalty for very short/long matches
        let content_length = result.vector.end_line - result.vector.start_line;
        if content_length < 5 {
            adjusted_score *= 0.9; // Penalize very short snippets
        } else if content_length > 100 {
            adjusted_score *= 0.8; // Penalize very long snippets
        }

        adjusted_score.min(1.0)
    }

    pub fn build_index(&mut self) -> Result<()> {
        self.index.optimize_search()?;
        Ok(())
    }

    pub fn size(&self) -> usize {
        self.index.size()
    }
}

// Performance monitoring
#[derive(Debug, Clone)]
pub struct SearchMetrics {
    pub query_time_ms: u64,
    pub candidates_considered: usize,
    pub vectors_searched: usize,
    pub cache_hit_rate: f32,
}

impl SearchMetrics {
    pub fn new() -> Self {
        Self {
            query_time_ms: 0,
            candidates_considered: 0,
            vectors_searched: 0,
            cache_hit_rate: 0.0,
        }
    }
}