// ABOUTME: Model registry and metadata management

use crate::error::{CodeSearchError, Result};
use crate::models::ModelMetadata;
use std::collections::HashMap;

pub struct ModelRegistry {
    models: HashMap<String, ModelMetadata>,
}

impl ModelRegistry {
    pub fn new() -> Self {
        let mut registry = Self {
            models: HashMap::new(),
        };

        // Register built-in models
        registry.register_builtin_models();
        registry
    }

    pub fn register_model(&mut self, metadata: ModelMetadata) {
        self.models.insert(metadata.name.clone(), metadata);
    }

    pub fn get_model_metadata(&self, model_name: &str) -> Result<&ModelMetadata> {
        self.models.get(model_name)
            .ok_or_else(|| CodeSearchError::Model(format!(
                "Model not found: {}",
                model_name
            )))
    }

    pub fn list_models(&self) -> Vec<&ModelMetadata> {
        self.models.values().collect()
    }

    pub fn get_models_for_language(&self, language: &str) -> Vec<&ModelMetadata> {
        self.models.values()
            .filter(|metadata| metadata.languages.contains(&language.to_string()))
            .collect()
    }

    pub fn is_model_available(&self, model_name: &str) -> bool {
        self.models.contains_key(model_name)
    }

    fn register_builtin_models(&mut self) {
        // Register CodeBERT Small
        self.register_model(ModelMetadata {
            name: "codebert-small".to_string(),
            version: "1.0.0".to_string(),
            embedding_dimension: 768,
            max_sequence_length: 512,
            model_type: "transformer".to_string(),
            languages: vec![
                "rust".to_string(),
                "python".to_string(),
                "javascript".to_string(),
                "typescript".to_string(),
                "java".to_string(),
                "go".to_string(),
                "c".to_string(),
                "cpp".to_string(),
            ],
        });

        // Register a small test model for development
        self.register_model(ModelMetadata {
            name: "test-model".to_string(),
            version: "0.1.0".to_string(),
            embedding_dimension: 128,
            max_sequence_length: 256,
            model_type: "mock".to_string(),
            languages: vec![
                "rust".to_string(),
                "python".to_string(),
                "markdown".to_string(),
            ],
        });
    }

    pub fn get_default_model(&self) -> Result<&ModelMetadata> {
        self.get_model_metadata("codebert-small")
            .or_else(|_| self.get_model_metadata("test-model"))
    }

    pub fn get_recommended_model_for_language(&self, language: &str) -> Result<&ModelMetadata> {
        let models_for_language = self.get_models_for_language(language);

        // Prefer CodeBERT for programming languages
        for model in models_for_language {
            if model.name.contains("codebert") {
                return Ok(model);
            }
        }

        // Fall back to test model
        self.get_model_metadata("test-model")
    }
}

impl Default for ModelRegistry {
    fn default() -> Self {
        Self::new()
    }
}