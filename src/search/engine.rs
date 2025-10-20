// ABOUTME: Main search orchestration for codesearch

use crate::error::Result;
use crate::models::embeddings::cosine_similarity;
use crate::storage::{MetadataStorage, VectorStorage, IndexStatistics};
use crate::search::vector_index::{ANNIndex, IndexedVector, SearchResult as VectorResult, SearchMetrics};
use std::path::{Path, PathBuf};
use std::time::Instant;

#[derive(Debug, Clone)]
pub struct SearchQuery {
    pub text: String,
    pub query_type: QueryType,
    pub filters: SearchFilters,
    pub limit: usize,
}

#[derive(Debug, Clone)]
pub enum QueryType {
    Semantic,
    Symbol,
    Hybrid,
}

#[derive(Debug, Clone)]
pub struct SearchFilters {
    pub file_types: Vec<String>,
    pub symbol_types: Vec<String>,
    pub min_score: f32,
    pub exclude_patterns: Vec<String>,
}

#[derive(Debug, Clone)]
pub struct SearchResult {
    pub file_path: PathBuf,
    pub start_line: usize,
    pub end_line: usize,
    pub score: f32,
    pub result_type: SearchResultType,
    pub context: String,
    pub code_snippet: String,
    pub symbols: Vec<String>,
    pub chunk_type: String,
}

#[derive(Debug, Clone)]
pub enum SearchResultType {
    SemanticMatch(f32),
    ExactSymbolMatch,
    FuzzySymbolMatch(f32),
    HybridMatch(f32),
}

pub struct SearchEngine {
    project_root: PathBuf,
    metadata_storage: MetadataStorage,
    vector_storage: VectorStorage,
    vector_index: ANNIndex,
}

impl SearchEngine {
    pub fn new<P: AsRef<Path>>(project_root: P) -> Result<Self> {
        let project_root = project_root.as_ref().to_path_buf();
        let index_dir = project_root.join(".codesearch");

        // Initialize storage components
        let metadata_storage = MetadataStorage::new(index_dir.join("metadata.db"))?;
        let vector_storage = VectorStorage::open(index_dir.join("vectors.dat"))?;
        let vector_index = ANNIndex::new(768, 100000); // 768 dimensions, max 100k vectors

        Ok(Self {
            project_root,
            metadata_storage,
            vector_storage,
            vector_index,
        })
    }

    pub async fn search(&mut self, query: &SearchQuery) -> Result<Vec<SearchResult>> {
        // Get project metadata
        let project = self.metadata_storage.get_project_by_path(&self.project_root)?
            .ok_or_else(|| crate::error::CodeSearchError::Search(
                "Project not found in index".to_string()
            ))?;

        match query.query_type {
            QueryType::Semantic => self.semantic_search(project.id, query).await,
            QueryType::Symbol => self.symbol_search(project.id, query).await,
            QueryType::Hybrid => self.hybrid_search(project.id, query).await,
        }
    }

    async fn semantic_search(&mut self, project_id: i64, query: &SearchQuery) -> Result<Vec<SearchResult>> {
        // Generate query embedding using mock model
        let query_embedding = self.generate_query_embedding(&query.text)?;

        // Get candidate chunks from database
        let candidates = self.metadata_storage.get_chunks_for_search(project_id)?;

        let mut results = Vec::new();

        for candidate in candidates {
            // Apply filters
            if !self.passes_filters(&candidate, &query.filters) {
                continue;
            }

            // Get vector for this chunk
            let chunk_embedding = self.vector_storage.get_vector(candidate.5 as u32)?;

            // Calculate similarity
            let similarity = cosine_similarity(&query_embedding, &chunk_embedding);

            if similarity >= query.filters.min_score {
                // Extract context and code snippet
                let (context, code_snippet) = self.extract_content(&candidate.1, candidate.2, candidate.3)?;

                results.push(SearchResult {
                    file_path: candidate.1,
                    start_line: candidate.2,
                    end_line: candidate.3,
                    score: similarity,
                    result_type: SearchResultType::SemanticMatch(similarity),
                    context,
                    code_snippet,
                    symbols: vec![], // Will be populated later
                    chunk_type: candidate.4,
                });
            }
        }

        // Sort by score and apply limit
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        results.truncate(query.limit);

        Ok(results)
    }

    async fn symbol_search(&mut self, project_id: i64, query: &SearchQuery) -> Result<Vec<SearchResult>> {
        // Search for exact symbol matches
        let symbol_matches = self.metadata_storage.find_symbols_by_name(project_id, &query.text)?;

        let mut results = Vec::new();

        for (symbol_name, file_path, start_line, end_line) in symbol_matches {
            // Apply filters
            if !self.passes_file_type_filter(&file_path, &query.filters.file_types) {
                continue;
            }

            // Extract context and code snippet
            let (context, code_snippet) = self.extract_content(&file_path, start_line, end_line)?;

            // Determine if this is an exact or fuzzy match
            let is_exact = symbol_name.to_lowercase() == query.text.to_lowercase();
            let fuzzy_score = if is_exact { 1.0 } else { self.calculate_fuzzy_score(&symbol_name, &query.text) };

            if fuzzy_score >= query.filters.min_score {
                results.push(SearchResult {
                    file_path,
                    start_line,
                    end_line,
                    score: fuzzy_score,
                    result_type: if is_exact {
                        SearchResultType::ExactSymbolMatch
                    } else {
                        SearchResultType::FuzzySymbolMatch(fuzzy_score)
                    },
                    context,
                    code_snippet,
                    symbols: vec![symbol_name.clone()],
                    chunk_type: "Symbol".to_string(),
                });
            }
        }

        // Sort and limit results
        results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        results.truncate(query.limit);

        Ok(results)
    }

    async fn hybrid_search(&mut self, project_id: i64, query: &SearchQuery) -> Result<Vec<SearchResult>> {
        // Run semantic search first
        let semantic_results = self.semantic_search(project_id, query).await?;

        // Then run symbol search
        let symbol_results = self.symbol_search(project_id, query).await?;

        // Combine and deduplicate results
        let mut combined_results = semantic_results;
        combined_results.extend(symbol_results);

        // Remove duplicates (same file and line range)
        combined_results.sort_by(|a, b| a.file_path.cmp(&b.file_path)
            .then(a.start_line.cmp(&b.start_line)));
        combined_results.dedup_by(|a, b| a.file_path == b.file_path
            && a.start_line == b.start_line
            && a.end_line == b.end_line);

        // Apply hybrid scoring
        for result in &mut combined_results {
            result.score = self.calculate_hybrid_score(result);
            result.result_type = SearchResultType::HybridMatch(result.score);
        }

        // Sort and limit
        combined_results.sort_by(|a, b| b.score.partial_cmp(&a.score).unwrap());
        combined_results.truncate(query.limit);

        Ok(combined_results)
    }

    fn passes_filters(&self, candidate: &(i64, PathBuf, usize, usize, String, i64), filters: &SearchFilters) -> bool {
        // Check file type filter
        if !self.passes_file_type_filter(&candidate.1, &filters.file_types) {
            return false;
        }

        // Check exclude patterns
        if !filters.exclude_patterns.is_empty() {
            let path_str = candidate.1.to_string_lossy();
            if filters.exclude_patterns.iter().any(|pattern| path_str.contains(pattern)) {
                return false;
            }
        }

        true
    }

    fn passes_file_type_filter(&self, file_path: &Path, file_types: &[String]) -> bool {
        if file_types.is_empty() {
            return true;
        }

        if let Some(extension) = file_path.extension().and_then(|ext| ext.to_str()) {
            file_types.iter().any(|ft| ft == extension)
        } else {
            false
        }
    }

    fn extract_content(&self, file_path: &Path, start_line: usize, end_line: usize) -> Result<(String, String)> {
        let content = std::fs::read_to_string(file_path)?;
        let lines: Vec<&str> = content.lines().collect();

        let context_start = start_line.saturating_sub(3);
        let context_end = end_line.min(lines.len()).saturating_add(2).min(lines.len());

        let context_lines = &lines[context_start..context_end];
        let context = context_lines.join("\n");

        let snippet_lines = &lines[start_line - 1..end_line.min(lines.len())];
        let code_snippet = snippet_lines.join("\n");

        Ok((context, code_snippet))
    }

    fn calculate_hybrid_score(&self, result: &SearchResult) -> f32 {
        match &result.result_type {
            SearchResultType::SemanticMatch(similarity) => similarity * 0.7, // Weight semantic search lower
            SearchResultType::ExactSymbolMatch => 1.0,
            SearchResultType::FuzzySymbolMatch(similarity) => similarity * 0.8,
            SearchResultType::HybridMatch(score) => *score,
        }
    }

    fn calculate_fuzzy_score(&self, symbol_name: &str, query: &str) -> f32 {
        let symbol_lower = symbol_name.to_lowercase();
        let query_lower = query.to_lowercase();

        if symbol_lower == query_lower {
            return 1.0;
        }

        if symbol_lower.contains(&query_lower) {
            return 0.8;
        }

        if query_lower.contains(&symbol_lower) {
            return 0.6;
        }

        // Simple Levenshtein distance approximation
        let distance = self.levenshtein_distance(&symbol_lower, &query_lower);
        let max_len = symbol_lower.len().max(query_lower.len());
        if max_len == 0 {
            return 1.0;
        }

        let similarity = 1.0 - (distance as f32 / max_len as f32);
        similarity.max(0.0)
    }

    fn levenshtein_distance(&self, s1: &str, s2: &str) -> usize {
        let len1 = s1.chars().count();
        let len2 = s2.chars().count();

        if len1 == 0 { return len2; }
        if len2 == 0 { return len1; }

        let mut matrix = vec![vec![0; len2 + 1]; len1 + 1];

        for i in 0..=len1 {
            matrix[i][0] = i;
        }
        for j in 0..=len2 {
            matrix[0][j] = j;
        }

        let chars1: Vec<char> = s1.chars().collect();
        let chars2: Vec<char> = s2.chars().collect();

        for i in 1..=len1 {
            for j in 1..=len2 {
                let cost = if chars1[i-1] == chars2[j-1] { 0 } else { 1 };
                matrix[i][j] = *[
                    matrix[i-1][j] + 1,           // deletion
                    matrix[i][j-1] + 1,           // insertion
                    matrix[i-1][j-1] + cost        // substitution
                ].iter().min().unwrap();
            }
        }

        matrix[len1][len2]
    }

    fn generate_query_embedding(&self, query: &str) -> Result<Vec<f32>> {
        // Mock embedding generation - will use actual model in production
        let mut embedding = Vec::with_capacity(768);
        let hash = self.simple_hash(query);

        for i in 0..768 {
            let value = ((hash.wrapping_mul(i as u64 + 1)) % 1000) as f32 / 1000.0;
            embedding.push(value);
        }

        Ok(embedding)
    }

    fn simple_hash(&self, text: &str) -> u64 {
        let mut hash = 0u64;
        for byte in text.as_bytes() {
            hash = hash.wrapping_mul(31).wrapping_add(*byte as u64);
        }
        hash
    }

    pub fn get_statistics(&self) -> Result<IndexStatistics> {
        let project = self.metadata_storage.get_project_by_path(&self.project_root)?
            .ok_or_else(|| crate::error::CodeSearchError::Search(
                "Project not found in index".to_string()
            ))?;

        self.metadata_storage.get_statistics(project.id)
    }

    pub fn is_indexed(&self) -> bool {
        self.metadata_storage.get_project_by_path(&self.project_root).is_ok()
    }
}

impl Default for SearchQuery {
    fn default() -> Self {
        Self {
            text: String::new(),
            query_type: QueryType::Hybrid,
            filters: SearchFilters::default(),
            limit: 20,
        }
    }
}

impl Default for SearchFilters {
    fn default() -> Self {
        Self {
            file_types: Vec::new(),
            symbol_types: Vec::new(),
            min_score: 0.5,
            exclude_patterns: Vec::new(),
        }
    }
}