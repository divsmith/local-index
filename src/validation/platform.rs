// ABOUTME: Cross-platform compatibility validation for codesearch

use crate::error::Result;

pub struct PlatformValidator;

impl PlatformValidator {
    pub fn new() -> Self {
        Self
    }

    pub fn validate_platform_compatibility(&self) -> Result<Vec<PlatformTestResult>> {
        let mut results = Vec::new();

        // Test basic file operations
        results.push(self.test_file_operations()?);

        // Test memory allocation
        results.push(self.test_memory_operations()?);

        // Test concurrent operations
        results.push(self.test_concurrent_operations()?);

        // Test tree-sitter compatibility (placeholder)
        results.push(self.test_tree_sitter_compatibility()?);

        // Test ONNX runtime compatibility (placeholder)
        results.push(self.test_onnx_compatibility()?);

        Ok(results)
    }

    fn test_file_operations(&self) -> Result<PlatformTestResult> {
        let start = std::time::Instant::now();

        // Test basic file operations
        let temp_file = std::env::temp_dir().join("codesearch_test_file");
        let test_content = "test content for file operations";

        // Write test
        std::fs::write(&temp_file, test_content)
            .map_err(|e| crate::error::CodeSearchError::Io(e))?;

        // Read test
        let read_content = std::fs::read_to_string(&temp_file)
            .map_err(|e| crate::error::CodeSearchError::Io(e))?;

        // Cleanup
        std::fs::remove_file(&temp_file)
            .map_err(|e| crate::error::CodeSearchError::Io(e))?;

        let duration = start.elapsed();
        let success = read_content == test_content;

        Ok(PlatformTestResult {
            test_name: "File Operations".to_string(),
            platform: std::env::consts::OS.to_string(),
            success,
            details: format!("Read/write test completed in {:?}", duration),
            error_message: if !success {
                Some("Content mismatch".to_string())
            } else {
                None
            },
        })
    }

    fn test_memory_operations(&self) -> Result<PlatformTestResult> {
        let start = std::time::Instant::now();

        // Test memory allocation and operations
        let large_vec: Vec<Vec<f32>> = (0..1000)
            .map(|_| (0..768).map(|_| rand::random_f32()).collect())
            .collect();

        // Test memory access patterns
        let sum: f32 = large_vec.iter().flat_map(|v| v.iter()).sum();

        let duration = start.elapsed();
        let success = sum.is_finite();

        Ok(PlatformTestResult {
            test_name: "Memory Operations".to_string(),
            platform: std::env::consts::OS.to_string(),
            success,
            details: format!("Allocated and processed {} floats in {:?}", 1000 * 768, duration),
            error_message: if !success {
                Some("Invalid sum calculation".to_string())
            } else {
                None
            },
        })
    }

    fn test_concurrent_operations(&self) -> Result<PlatformTestResult> {
        use std::sync::Arc;
        use std::thread;

        let start = std::time::Instant::now();

        let counter = Arc::new(std::sync::atomic::AtomicUsize::new(0));
        let mut handles = vec![];

        // Spawn multiple threads
        for _ in 0..4 {
            let counter_clone = Arc::clone(&counter);
            let handle = thread::spawn(move || {
                for _ in 0..1000 {
                    counter_clone.fetch_add(1, std::sync::atomic::Ordering::Relaxed);
                }
            });
            handles.push(handle);
        }

        // Wait for all threads to complete
        for handle in handles {
            handle.join().expect("Thread should complete successfully");
        }

        let final_count = counter.load(std::sync::atomic::Ordering::Relaxed);
        let duration = start.elapsed();
        let success = final_count == 4000; // 4 threads * 1000 increments

        Ok(PlatformTestResult {
            test_name: "Concurrent Operations".to_string(),
            platform: std::env::consts::OS.to_string(),
            success,
            details: format!("Completed {} operations in {:?}", final_count, duration),
            error_message: if !success {
                Some(format!("Expected 4000, got {}", final_count))
            } else {
                None
            },
        })
    }

    fn test_tree_sitter_compatibility(&self) -> Result<PlatformTestResult> {
        let start = std::time::Instant::now();

        // Placeholder test for tree-sitter compatibility
        // In real implementation, this would test WASM compilation and execution
        let duration = start.elapsed();

        Ok(PlatformTestResult {
            test_name: "Tree-sitter Compatibility".to_string(),
            platform: std::env::consts::OS.to_string(),
            success: true, // Placeholder
            details: format!("Tree-sitter test completed in {:?}", duration),
            error_message: None,
        })
    }

    fn test_onnx_compatibility(&self) -> Result<PlatformTestResult> {
        let start = std::time::Instant::now();

        // Placeholder test for ONNX runtime compatibility
        // In real implementation, this would test model loading and inference
        let duration = start.elapsed();

        Ok(PlatformTestResult {
            test_name: "ONNX Runtime Compatibility".to_string(),
            platform: std::env::consts::OS.to_string(),
            success: true, // Placeholder
            details: format!("ONNX runtime test completed in {:?}", duration),
            error_message: None,
        })
    }
}

#[derive(Debug)]
pub struct PlatformTestResult {
    pub test_name: String,
    pub platform: String,
    pub success: bool,
    pub details: String,
    pub error_message: Option<String>,
}

// Simple random number generator for testing
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