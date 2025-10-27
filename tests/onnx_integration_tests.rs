// ABOUTME: Integration tests for ONNX embedding functionality

use codesearch::models::{ModelManager, ModelConfig};
use codesearch::error::Result;

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_fallback_embeddings() {
        // Test that fallback embeddings work without ONNX dependencies
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_texts = vec![
            "fn test_function() { println!(\"Hello, world!\"); }".to_string(),
            "class TestClass: pass".to_string(),
            "const MAX_SIZE: usize = 1024;".to_string(),
        ];

        // This should work even without ONNX models available
        let embeddings = manager.generate_embeddings(&test_texts).await
            .expect("Failed to generate fallback embeddings");

        assert_eq!(embeddings.len(), test_texts.len());
        for embedding in embeddings {
            assert_eq!(embedding.len(), 768); // Default embedding dimension
        }
    }

    #[tokio::test]
    async fn test_single_embedding_generation() {
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_code = "pub fn fibonacci(n: u32) -> u64 {
            match n {
                0 => 0,
                1 => 1,
                _ => fibonacci(n - 1) + fibonacci(n - 2),
            }
        }";

        let embedding = manager.generate_single_embedding(test_code).await
            .expect("Failed to generate single embedding");

        assert_eq!(embedding.len(), 768);
    }

    #[tokio::test]
    async fn test_sync_embeddings() {
        let config = ModelConfig::default();
        let manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_texts = vec![
            "async fn fetch_data(url: &str) -> Result<String>".to_string(),
            "let x = 42;".to_string(),
        ];

        let embeddings = manager.generate_embeddings_sync(&test_texts)
            .expect("Failed to generate sync embeddings");

        assert_eq!(embeddings.len(), test_texts.len());
        for embedding in embeddings {
            assert_eq!(embedding.len(), 768);
        }
    }

    #[tokio::test]
    async fn test_embedding_consistency() {
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_text = "fn hello_world() { println!(\"Hello, world!\"); }";

        // Generate embeddings multiple times
        let embedding1 = manager.generate_single_embedding(test_text).await.unwrap();
        let embedding2 = manager.generate_single_embedding(test_text).await.unwrap();

        // Should be identical (deterministic)
        assert_eq!(embedding1, embedding2);
    }

    #[tokio::test]
    async fn test_different_code_types() {
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        let code_samples = vec![
            "fn function() {}", // Function
            "struct MyStruct {}", // Struct
            "impl MyStruct { fn new() -> Self { Self {} } }", // Implementation
            "use std::collections::HashMap;", // Import
            "const PI: f64 = 3.14159;", // Constant
            "if condition { do_something() }", // Control flow
            "for item in items { process(item) }", // Loop
        ];

        let embeddings = manager.generate_embeddings(&code_samples).await
            .expect("Failed to generate embeddings for different code types");

        assert_eq!(embeddings.len(), code_samples.len());

        // All embeddings should have the same dimension
        let expected_dim = manager.get_config().embedding_dimension;
        for embedding in embeddings {
            assert_eq!(embedding.len(), expected_dim);
        }
    }

    #[test]
    fn test_model_config() {
        let config = ModelConfig::default();
        assert_eq!(config.embedding_dimension, 768);
        assert_eq!(config.max_sequence_length, 512);
        assert_eq!(config.default_model, "semantic-mock-v1");
        assert_eq!(config.cache_size, 1000);
    }

    #[test]
    fn test_empty_embeddings() {
        let config = ModelConfig::default();
        let manager = ModelManager::new(config).expect("Failed to create model manager");

        let empty_texts: Vec<String> = vec![];
        let embeddings = manager.generate_embeddings_sync(&empty_texts)
            .expect("Failed to handle empty input");

        assert!(embeddings.is_empty());
    }

    #[test]
    fn test_large_text_handling() {
        let config = ModelConfig::default();
        let manager = ModelManager::new(config).expect("Failed to create model manager");

        // Create a very large text to test truncation handling
        let large_text = "fn large_function() { ".to_string() + &"let x = 42; ".repeat(1000) + "}";

        let embeddings = manager.generate_embeddings_sync(&[large_text])
            .expect("Failed to handle large text");

        assert_eq!(embeddings.len(), 1);
        assert_eq!(embeddings[0].len(), 768);
    }
}

#[cfg(test)]
mod onnx_specific_tests {
    use super::*;

    // These tests will only run when ONNX models are available
    #[tokio::test]
    #[ignore] // Requires actual ONNX model files
    async fn test_onnx_model_loading() {
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        // This test requires actual ONNX model files to be present
        // In a real environment, the model would be downloaded automatically
        let test_text = "fn test() -> u32 { 42 }";

        let _embedding = manager.generate_single_embedding(test_text).await
            .expect("Failed to generate ONNX embedding");
    }

    #[tokio::test]
    #[ignore] // Requires actual ONNX model files
    async fn test_onnx_batch_processing() {
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_texts = vec![
            "fn function_a() {}".to_string(),
            "fn function_b() {}".to_string(),
            "fn function_c() {}".to_string(),
        ];

        let embeddings = manager.generate_embeddings(&test_texts).await
            .expect("Failed to generate ONNX batch embeddings");

        assert_eq!(embeddings.len(), test_texts.len());
    }
}

// Benchmark tests for performance validation
#[cfg(test)]
mod benchmarks {
    use super::*;
    use std::time::Instant;

    #[tokio::test]
    async fn benchmark_embedding_generation() {
        let config = ModelConfig::default();
        let mut manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_texts: Vec<String> = (0..100)
            .map(|i| format!("fn test_{}() -> u32 {{ {} }}", i, i))
            .collect();

        let start = Instant::now();
        let embeddings = manager.generate_embeddings(&test_texts).await
            .expect("Failed to generate embeddings for benchmark");
        let duration = start.elapsed();

        assert_eq!(embeddings.len(), test_texts.len());
        println!("Generated {} embeddings in {:?}", embeddings.len(), duration);

        // Performance should be reasonable (less than 1 second for 100 texts)
        assert!(duration.as_secs() < 1, "Embedding generation took too long: {:?}", duration);
    }

    #[test]
    fn benchmark_sync_embedding_generation() {
        let config = ModelConfig::default();
        let manager = ModelManager::new(config).expect("Failed to create model manager");

        let test_texts: Vec<String> = (0..1000)
            .map(|i| format!("let x_{} = {};", i, i))
            .collect();

        let start = Instant::now();
        let embeddings = manager.generate_embeddings_sync(&test_texts)
            .expect("Failed to generate sync embeddings for benchmark");
        let duration = start.elapsed();

        assert_eq!(embeddings.len(), test_texts.len());
        println!("Generated {} sync embeddings in {:?}", embeddings.len(), duration);

        // Sync version should be fast (less than 100ms for 1000 texts)
        assert!(duration.as_millis() < 100, "Sync embedding generation took too long: {:?}", duration);
    }
}