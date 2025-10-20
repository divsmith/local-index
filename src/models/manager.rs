// ABOUTME: Model loading and management

use crate::error::{CodeSearchError, Result};
use crate::models::{ModelConfig, LoadedModel, ModelMetadata};
use std::collections::HashMap;
use std::sync::{Arc, Mutex};

#[derive(Clone)]
pub struct ModelManager {
    loaded_models: Arc<Mutex<HashMap<String, LoadedModel>>>,
    config: ModelConfig,
}

impl ModelManager {
    pub fn new(config: ModelConfig) -> Result<Self> {
        Ok(Self {
            loaded_models: Arc::new(Mutex::new(HashMap::new())),
            config,
        })
    }

    pub fn generate_embeddings(&self, code_snippets: &[String]) -> Result<Vec<Vec<f32>>> {
        let model_name = &self.config.default_model;
        let mut models = self.loaded_models.lock().unwrap();

        let model = models.entry(model_name.clone())
            .or_insert_with(|| self.load_model(model_name).unwrap());

        // Generate embeddings for each code snippet
        let mut embeddings = Vec::with_capacity(code_snippets.len());
        for snippet in code_snippets {
            let embedding = model.generate_mock_embedding(snippet);
            embeddings.push(embedding);
        }

        Ok(embeddings)
    }

    pub fn load_model(&self, model_name: &str) -> Result<LoadedModel> {
        // In a real implementation, this would load ONNX models from embedded data or download them
        // For now, we'll create a mock model
        match model_name {
            "codebert-small" => {
                let model = LoadedModel::new(
                    model_name.to_string(),
                    768, // embedding dimension
                );
                Ok(model)
            }
            _ => Err(CodeSearchError::Model(format!(
                "Unknown model: {}",
                model_name
            )))
        }
    }

    pub fn unload_model(&self, model_name: &str) -> Result<()> {
        let mut models = self.loaded_models.lock().unwrap();
        models.remove(model_name);
        Ok(())
    }

    pub fn get_model_metadata(&self, model_name: &str) -> Result<ModelMetadata> {
        match model_name {
            "codebert-small" => Ok(ModelMetadata::codebert_small()),
            _ => Err(CodeSearchError::Model(format!(
                "Unknown model: {}",
                model_name
            )))
        }
    }

    pub fn list_loaded_models(&self) -> Vec<String> {
        let models = self.loaded_models.lock().unwrap();
        models.keys().cloned().collect()
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