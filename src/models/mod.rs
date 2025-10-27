// ABOUTME: Model management for codesearch

use crate::error::{CodeSearchError, Result};

pub mod embeddings;
pub mod registry;
pub mod onnx_embeddings;
pub mod fallback;

pub use embeddings::{EmbeddingGenerator, CodeChunk};
pub use registry::ModelRegistry;
pub use onnx_embeddings::{OnnxEmbeddingGenerator as OnnxGenerator, ModelManager as OnnxManager};
pub use fallback::FallbackModelManager;

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

// ONNX-powered model manager with real semantic embeddings
pub struct ModelManager {
    config: ModelConfig,
    onnx_manager: OnnxManager,
    fallback_manager: FallbackModelManager,
    onnx_model: Option<std::sync::Arc<OnnxGenerator>>,
}

impl ModelManager {
    pub fn new(config: ModelConfig) -> Result<Self> {
        Ok(Self {
            config,
            onnx_manager: OnnxManager::new()?,
            fallback_manager: FallbackModelManager::new(config.clone()),
            onnx_model: None,
        })
    }

    pub async fn generate_embeddings(&mut self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        // Try ONNX first, fall back to mock if unavailable
        if self.onnx_model.is_none() {
            match self.onnx_manager.load_model("codebert-base").await {
                Ok(model) => {
                    self.onnx_model = Some(model);
                }
                Err(_) => {
                    // Fall back to mock implementation
                    return self.fallback_manager.generate_embeddings(texts);
                }
            }
        }

        if let Some(model) = &self.onnx_model {
            let mut embeddings = Vec::with_capacity(texts.len());
            for text in texts {
                let embedding = model.generate(text).await?;
                embeddings.push(embedding);
            }
            Ok(embeddings)
        } else {
            self.fallback_manager.generate_embeddings(texts)
        }
    }

    pub fn generate_embeddings_sync(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        self.fallback_manager.generate_embeddings(texts)
    }

    pub async fn generate_single_embedding(&mut self, text: &str) -> Result<Vec<f32>> {
        let mut results = self.generate_embeddings(&[text.to_string()]).await?;
        Ok(results.remove(0))
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