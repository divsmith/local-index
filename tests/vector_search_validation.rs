// ABOUTME: Performance validation tests for vector search functionality

use codesearch::validation::PerformanceValidator;
use codesearch::utils::math::*;
use std::time::Instant;

#[test]
fn test_small_index_performance() {
    let validator = PerformanceValidator::new();

    // Start timer
    let start = Instant::now();

    // Run performance validation
    let results = validator.validate_vector_search_performance().unwrap();

    let duration = start.elapsed();

    // Find the small index test result
    let small_test = results.iter()
        .find(|r| r.test_name.contains("Small Index"))
        .expect("Should have small index performance test");

    assert!(small_test.success, "Small index performance test should pass: {}", small_test.details);
    assert!(duration.as_millis() < 200, "Overall validation should complete quickly, got {:?}", duration);

    println!("✅ Small index performance test passed: {}", small_test.details);
}

#[test]
fn test_medium_index_performance() {
    let validator = PerformanceValidator::new();

    // Start timer
    let start = Instant::now();

    // Run performance validation
    let results = validator.validate_vector_search_performance().unwrap();

    let duration = start.elapsed();

    // Find the medium index test result
    let medium_test = results.iter()
        .find(|r| r.test_name.contains("Medium Index"))
        .expect("Should have medium index performance test");

    assert!(medium_test.success, "Medium index performance test should pass: {}", medium_test.details);
    assert!(duration.as_millis() < 500, "Overall validation should complete in reasonable time, got {:?}", duration);

    println!("✅ Medium index performance test passed: {}", medium_test.details);
}

#[test]
fn test_memory_usage() {
    let validator = PerformanceValidator::new();

    // Start timer
    let start = Instant::now();

    // Run performance validation
    let results = validator.validate_vector_search_performance().unwrap();

    let duration = start.elapsed();

    // Find the memory usage test result
    let memory_test = results.iter()
        .find(|r| r.test_name.contains("Memory Usage"))
        .expect("Should have memory usage test");

    assert!(memory_test.success, "Memory usage test should pass: {}", memory_test.details);
    assert!(duration.as_millis() < 1000, "Memory test should complete in reasonable time, got {:?}", duration);

    println!("✅ Memory usage test passed: {}", memory_test.details);
}

#[test]
fn test_performance_targets() {
    let validator = PerformanceValidator::new();
    let results = validator.validate_vector_search_performance().unwrap();

    // Check that all tests pass
    for result in &results {
        assert!(result.success, "Performance test '{}' should pass: {}", result.test_name, result.details);

        // Check specific performance targets based on test type
        if result.test_name.contains("Small Index") {
            assert!(result.duration_ms < 50, "Small index search should be <50ms, got {}ms", result.duration_ms);
        } else if result.test_name.contains("Medium Index") {
            assert!(result.duration_ms < 100, "Medium index search should be <100ms, got {}ms", result.duration_ms);
        }
    }

    println!("✅ All performance targets met");
}

#[test]
fn test_cosine_similarity_accuracy() {
    // Test identical vectors
    let v1 = vec![1.0, 2.0, 3.0];
    let v2 = vec![1.0, 2.0, 3.0];
    let similarity = cosine_similarity(&v1, &v2);
    assert!((similarity - 1.0).abs() < 1e-6, "Identical vectors should have similarity 1.0, got {}", similarity);

    // Test orthogonal vectors
    let v3 = vec![1.0, 0.0];
    let v4 = vec![0.0, 1.0];
    let similarity2 = cosine_similarity(&v3, &v4);
    assert!((similarity2 - 0.0).abs() < 1e-6, "Orthogonal vectors should have similarity 0.0, got {}", similarity2);

    // Test opposite vectors
    let v5 = vec![1.0, 2.0, 3.0];
    let v6 = vec![-1.0, -2.0, -3.0];
    let similarity3 = cosine_similarity(&v5, &v6);
    assert!((similarity3 + 1.0).abs() < 1e-6, "Opposite vectors should have similarity -1.0, got {}", similarity3);

    println!("✅ Cosine similarity accuracy tests passed");
}

#[test]
fn test_vector_generation_consistency() {
    let validator = PerformanceValidator::new();

    // Generate vectors twice and ensure consistency in properties
    let vectors1 = validator.generate_test_vectors(100, 768).unwrap();
    let vectors2 = validator.generate_test_vectors(100, 768).unwrap();

    assert_eq!(vectors1.len(), 100, "Should generate 100 vectors");
    assert_eq!(vectors2.len(), 100, "Should generate 100 vectors");

    // Check dimensions
    for vector in &vectors1 {
        assert_eq!(vector.len(), 768, "Each vector should have 768 dimensions");
    }

    // Check value ranges (should be between -1.0 and 1.0)
    for vector in &vectors1 {
        for &value in vector {
            assert!(value >= -1.0 && value <= 1.0, "Vector values should be in [-1.0, 1.0], got {}", value);
        }
    }

    println!("✅ Vector generation consistency tests passed");
}