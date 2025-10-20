// ABOUTME: Performance benchmarks for vector search operations

use criterion::{black_box, criterion_group, criterion_main, Criterion};
use codesearch::validation::PerformanceValidator;
use codesearch::utils::math::*;

fn benchmark_small_vector_search(c: &mut Criterion) {
    let validator = PerformanceValidator::new();
    let vectors = validator.generate_test_vectors(1000, 768).unwrap();
    let query = validator.generate_query_vector(768).unwrap();

    c.bench_function("search_1000_vectors", |b| {
        b.iter(|| {
            let results = search_similar_internal(
                black_box(&query),
                black_box(&vectors),
                black_box(10)
            ).unwrap();
            black_box(results);
        })
    });
}

fn benchmark_medium_vector_search(c: &mut Criterion) {
    let validator = PerformanceValidator::new();
    let vectors = validator.generate_test_vectors(10000, 768).unwrap();
    let query = validator.generate_query_vector(768).unwrap();

    c.bench_function("search_10000_vectors", |b| {
        b.iter(|| {
            let results = search_similar_internal(
                black_box(&query),
                black_box(&vectors),
                black_box(10)
            ).unwrap();
            black_box(results);
        })
    });
}

fn benchmark_cosine_similarity(c: &mut Criterion) {
    let v1: Vec<f32> = (0..768).map(|i| i as f32 / 1000.0).collect();
    let v2: Vec<f32> = (0..768).map(|i| (i * 2) as f32 / 1000.0).collect();

    c.bench_function("cosine_similarity_768d", |b| {
        b.iter(|| {
            let similarity = cosine_similarity(black_box(&v1), black_box(&v2));
            black_box(similarity);
        })
    });
}

fn benchmark_vector_generation(c: &mut Criterion) {
    let validator = PerformanceValidator::new();

    c.bench_function("generate_100_vectors_768d", |b| {
        b.iter(|| {
            let vectors = validator.generate_test_vectors(black_box(100), black_box(768)).unwrap();
            black_box(vectors);
        })
    });
}

fn benchmark_memory_allocation(c: &mut Criterion) {
    c.bench_function("allocate_50k_vectors_768d", |b| {
        b.iter(|| {
            let vectors: Vec<Vec<f32>> = (0..black_box(50000))
                .map(|_| (0..768).map(|_| rand::random_f32()).collect())
                .collect();
            black_box(vectors);
        })
    });
}

criterion_group!(
    benches,
    benchmark_small_vector_search,
    benchmark_medium_vector_search,
    benchmark_cosine_similarity,
    benchmark_vector_generation,
    benchmark_memory_allocation
);

criterion_main!(benches);


fn search_similar_internal(
    query: &[f32],
    vectors: &[Vec<f32>],
    k: usize
) -> codesearch::error::Result<Vec<usize>> {
    let mut similarities: Vec<(usize, f32)> = vectors
        .iter()
        .enumerate()
        .map(|(i, v)| (i, cosine_similarity(query, v)))
        .collect();

    similarities.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap());

    Ok(similarities.into_iter().take(k).map(|(i, _)| i).collect())
}

// Simple random number generator for benchmarks
mod rand {
    use std::cell::Cell;
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};

    thread_local! {
        static RNG_STATE: Cell<u64> = Cell::new(99999);
    }

    pub fn random_f32() -> f32 {
        RNG_STATE.with(|state| {
            let current = state.get();
            let mut hasher = DefaultHasher::new();
            current.hash(&mut hasher);
            let next = hasher.finish();
            state.set(next);
            (next % 1000000) as f32 / 1000000.0 * 2.0 - 1.0
        })
    }
}