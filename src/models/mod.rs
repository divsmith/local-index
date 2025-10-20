// ABOUTME: Model management for codesearch

use crate::error::{CodeSearchError, Result};

pub mod embeddings;
pub mod registry;

pub use embeddings::{EmbeddingGenerator, CodeChunk};
pub use registry::ModelRegistry;

// Model configuration
#[derive(Debug, Clone)]
pub struct ModelConfig {
    pub default_model: String,
    pub max_sequence_length: usize,
    pub embedding_dimension: usize,
    pub cache_size: usize,
}

impl Default for ModelConfig {
    fn default() -> Self {
        Self {
            default_model: "semantic-mock-v1".to_string(),
            max_sequence_length: 512,
            embedding_dimension: 768,
            cache_size: 1000,
        }
    }
}

// Improved mock model manager with better semantic embeddings
#[derive(Clone)]
pub struct ModelManager {
    config: ModelConfig,
}

impl ModelManager {
    pub fn new(config: ModelConfig) -> Result<Self> {
        Ok(Self { config })
    }

    pub fn generate_embeddings(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        let mut embeddings = Vec::with_capacity(texts.len());

        for text in texts {
            let embedding = self.generate_semantic_embedding(text);
            embeddings.push(embedding);
        }

        Ok(embeddings)
    }

    fn generate_semantic_embedding(&self, text: &str) -> Vec<f32> {
        let mut embedding = Vec::with_capacity(self.config.embedding_dimension);

        // Use a more sophisticated embedding generation that captures semantic meaning
        let semantic_features = self.extract_semantic_features(text);

        for i in 0..self.config.embedding_dimension {
            let feature_value = if i < semantic_features.len() {
                semantic_features[i]
            } else {
                // Fill remaining dimensions with smoothed noise
                let hash = self.complex_hash(&format!("{}:{}", text, i));
                ((hash as f64 / u64::MAX as f64) * 2.0 - 1.0) as f32 * 0.1
            };
            embedding.push(feature_value);
        }

        // Normalize the embedding
        self.normalize_embedding(&mut embedding);
        embedding
    }

    fn extract_semantic_features(&self, text: &str) -> Vec<f32> {
        let mut features = Vec::new();
        let lowercase = text.to_lowercase();

        // Programming language indicators
        features.push(if lowercase.contains("fn") || lowercase.contains("func") || lowercase.contains("def") { 0.8 } else { 0.0 });
        features.push(if lowercase.contains("class") || lowercase.contains("struct") { 0.8 } else { 0.0 });
        features.push(if lowercase.contains("import") || lowercase.contains("use") { 0.7 } else { 0.0 });
        features.push(if lowercase.contains("test") || lowercase.contains("assert") { 0.6 } else { 0.0 });
        features.push(if lowercase.contains("error") || lowercase.contains("err") { 0.7 } else { 0.0 });
        features.push(if lowercase.contains("async") || lowercase.contains("await") { 0.6 } else { 0.0 });

        // Common programming concepts
        features.push(if lowercase.contains("loop") || lowercase.contains("for") || lowercase.contains("while") { 0.5 } else { 0.0 });
        features.push(if lowercase.contains("if") || lowercase.contains("match") { 0.4 } else { 0.0 });
        features.push(if lowercase.contains("return") || lowercase.contains("yield") { 0.5 } else { 0.0 });

        // Code-specific features
        features.push(if text.contains("=>") || text.contains("->") { 0.6 } else { 0.0 }); // Function arrows
        features.push(if text.contains("::") { 0.5 } else { 0.0 }); // Namespaces
        features.push(if text.contains("()") { 0.3 } else { 0.0 }); // Functions
        features.push(if text.contains("{}") { 0.4 } else { 0.0 }); // Blocks

        // Text characteristics
        features.push(text.len() as f32 / 1000.0); // Length feature
        features.push(lowercase.split_whitespace().count() as f32 / 100.0); // Word count
        features.push(lowercase.matches("r#").count() as f32 / 10.0); // Raw strings
        features.push(lowercase.matches("//").count() as f32 / 10.0); // Comments

        // Generate more features to reach a reasonable size
        while features.len() < 100 {
            let next_feature = features.len() as f32 / features.len() as f32;
            features.push(next_feature);
        }

        features
    }

    fn complex_hash(&self, text: &str) -> u64 {
        let mut hash = 0u64;
        let bytes = text.as_bytes();

        for (i, &byte) in bytes.iter().enumerate() {
            hash = hash.wrapping_mul(31).wrapping_add(byte as u64);
            hash = hash.wrapping_mul(i as u64 + 1);
        }

        hash
    }

    fn normalize_embedding(&self, embedding: &mut Vec<f32>) {
        let magnitude: f32 = embedding.iter().map(|x| x * x).sum::<f32>().sqrt();
        if magnitude > 0.0 {
            for value in embedding.iter_mut() {
                *value /= magnitude;
            }
        }
    }

    pub fn get_config(&self) -> &ModelConfig {
        &self.config
    }
}

impl Default for ModelManager {
    fn default() -> Self {
        Self::new(ModelConfig::default()).unwrap()
    }
}

#[derive(Debug, Clone)]
pub struct ModelMetadata {
    pub name: String,
    pub version: String,
    pub embedding_dimension: usize,
    pub max_sequence_length: usize,
    pub model_type: String,
    pub languages: Vec<String>,
}

impl ModelMetadata {
    pub fn semantic_mock_v1() -> Self {
        Self {
            name: "semantic-mock-v1".to_string(),
            version: "1.0.0".to_string(),
            embedding_dimension: 768,
            max_sequence_length: 512,
            model_type: "semantic-mock".to_string(),
            languages: vec!["rust".to_string(), "python".to_string(), "javascript".to_string()],
        }
    }
}