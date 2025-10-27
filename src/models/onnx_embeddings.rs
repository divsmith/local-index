// ABOUTME: ONNX-based embedding generation for real semantic code search

use crate::error::{CodeSearchError, Result};
use ndarray::Array1;
use ort::{Environment, Session, Value};
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::Arc;
use tokenizers::{Tokenizer, Encoding};

/// ONNX-based embedding generator using pre-trained code models
pub struct OnnxEmbeddingGenerator {
    session: Arc<Session>,
    tokenizer: Tokenizer,
    max_length: usize,
    embedding_dim: usize,
}

impl OnnxEmbeddingGenerator {
    /// Create a new ONNX embedding generator
    pub fn new(model_path: PathBuf, tokenizer_path: PathBuf) -> Result<Self> {
        // Set up ONNX Runtime environment
        let environment = Environment::builder()
            .with_name("codesearch")
            .with_execution_providers([
                ort::ExecutionProvider::cpu(),
                // ort::ExecutionProvider::cuda(), // Enable if GPU support needed
            ])
            .build()?
            .into_arc();

        // Load ONNX model
        let session = Session::builder(&environment)?
            .with_optimization_level(ort::GraphOptimizationLevel::Level1)?
            .with_model_from_file(model_path)?;

        // Load tokenizer
        let tokenizer = Tokenizer::from_file(tokenizer_path)
            .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to load tokenizer: {}", e)))?;

        // Get model info
        let input_shape = session.inputs[0].dimensions();
        let output_shape = session.outputs[0].dimensions();

        let max_length = input_shape[1] as usize;
        let embedding_dim = output_shape[1] as usize;

        Ok(Self {
            session: Arc::new(session),
            tokenizer: tokenizer.with_padding(None).with_truncation(None),
            max_length,
            embedding_dim,
        })
    }

    /// Generate embeddings for a batch of texts
    pub async fn generate_batch(&self, texts: &[String]) -> Result<Vec<Vec<f32>>> {
        if texts.is_empty() {
            return Ok(vec![]);
        }

        // Tokenize all texts
        let encodings: Vec<Encoding> = texts.iter()
            .map(|text| {
                self.tokenizer.encode(text, true)
                    .map_err(|e| CodeSearchError::EmbeddingError(format!("Tokenization failed: {}", e)))
            })
            .collect::<Result<Vec<_>>>()?;

        // Create batch tensors
        let batch_size = texts.len();
        let mut input_ids = Vec::with_capacity(batch_size * self.max_length);
        let mut attention_mask = Vec::with_capacity(batch_size * self.max_length);
        let mut token_type_ids = Vec::with_capacity(batch_size * self.max_length);

        for encoding in &encodings {
            let ids = encoding.get_ids();
            let mask = encoding.get_attention_mask();
            let type_ids = encoding.get_type_ids();

            // Pad or truncate to max_length
            for i in 0..self.max_length {
                if i < ids.len() {
                    input_ids.push(ids[i] as i64);
                    attention_mask.push(mask[i] as i64);
                    token_type_ids.push(type_ids[i] as i64);
                } else {
                    input_ids.push(0); // PAD token
                    attention_mask.push(0);
                    token_type_ids.push(0);
                }
            }
        }

        // Create input tensors
        let input_ids_shape = ndarray::IxDyn(&[batch_size, self.max_length]);
        let attention_mask_shape = ndarray::IxDyn(&[batch_size, self.max_length]);
        let token_type_ids_shape = ndarray::IxDyn(&[batch_size, self.max_length]);

        let input_ids_tensor = Value::from_array_view(
            &self.session.allocator,
            ndarray::ArrayView::from_shape(input_ids_shape, &input_ids)
                .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to create input_ids tensor: {}", e)))?,
        )?;

        let attention_mask_tensor = Value::from_array_view(
            &self.session.allocator,
            ndarray::ArrayView::from_shape(attention_mask_shape, &attention_mask)
                .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to create attention_mask tensor: {}", e)))?,
        )?;

        let token_type_ids_tensor = Value::from_array_view(
            &self.session.allocator,
            ndarray::ArrayView::from_shape(token_type_ids_shape, &token_type_ids)
                .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to create token_type_ids tensor: {}", e)))?,
        )?;

        // Run inference
        let outputs = self.session.run(vec![
            ("input_ids".into(), input_ids_tensor),
            ("attention_mask".into(), attention_mask_tensor),
            ("token_type_ids".into(), token_type_ids_tensor),
        ]).map_err(|e| CodeSearchError::EmbeddingError(format!("ONNX inference failed: {}", e)))?;

        // Extract embeddings
        let embeddings_tensor = outputs
            .get("last_hidden_state")
            .or_else(|| outputs.get("logits"))
            .ok_or_else(|| CodeSearchError::EmbeddingError("No embedding output found".to_string()))?;

        let embeddings_array = embeddings_tensor.extract_tensor::<f32>()
            .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to extract embeddings: {}", e)))?;

        // Convert tensor to Vec<Vec<f32>>
        let mut results = Vec::with_capacity(batch_size);

        for batch_idx in 0..batch_size {
            let mut embedding = Vec::with_capacity(self.embedding_dim);

            // For now, use the [CLS] token embedding (position 0) or mean pooling
            for dim_idx in 0..self.embedding_dim {
                let tensor_idx = batch_idx * self.max_length * self.embedding_dim + dim_idx;
                embedding.push(embeddings_array[tensor_idx]);
            }

            // Normalize the embedding
            self.normalize_embedding(&mut embedding);
            results.push(embedding);
        }

        Ok(results)
    }

    /// Generate embedding for a single text
    pub async fn generate(&self, text: &str) -> Result<Vec<f32>> {
        let mut results = self.generate_batch(&[text.to_string()]).await?;
        Ok(results.remove(0))
    }

    /// Get embedding dimension
    pub fn embedding_dimension(&self) -> usize {
        self.embedding_dim
    }

    /// Get max sequence length
    pub fn max_sequence_length(&self) -> usize {
        self.max_length
    }

    /// Normalize embedding vector
    fn normalize_embedding(&self, embedding: &mut [f32]) {
        let magnitude: f32 = embedding.iter().map(|x| x * x).sum::<f32>().sqrt();
        if magnitude > 0.0 {
            for value in embedding.iter_mut() {
                *value /= magnitude;
            }
        }
    }
}

/// Model downloader and cache manager
pub struct ModelManager {
    models: HashMap<String, Arc<OnnxEmbeddingGenerator>>,
    cache_dir: PathBuf,
}

impl ModelManager {
    pub fn new() -> Result<Self> {
        let cache_dir = std::env::var("HOME")
            .map(|home| PathBuf::from(home).join(".codesearch/models"))
            .unwrap_or_else(|_| PathBuf::from("/tmp/codesearch_models"));

        std::fs::create_dir_all(&cache_dir)
            .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to create model cache: {}", e)))?;

        Ok(Self {
            models: HashMap::new(),
            cache_dir,
        })
    }

    /// Load or download a model
    pub async fn load_model(&mut self, model_name: &str) -> Result<Arc<OnnxEmbeddingGenerator>> {
        if let Some(model) = self.models.get(model_name) {
            return Ok(Arc::clone(model));
        }

        let model_path = self.cache_dir.join(format!("{}.onnx", model_name));
        let tokenizer_path = self.cache_dir.join(format!("{}-tokenizer.json", model_name));

        // For now, we'll provide URLs for common code models
        let model_url = match model_name {
            "codebert-base" => "https://huggingface.co/microsoft/codebert-base/resolve/main/model.onnx",
            "unixcoder-base" => "https://huggingface.co/microsoft/unixcoder-base/resolve/main/model.onnx",
            "graphcodebert-base" => "https://huggingface.co/microsoft/graphcodebert-base/resolve/main/model.onnx",
            _ => return Err(CodeSearchError::EmbeddingError(format!("Unknown model: {}", model_name))),
        };

        let tokenizer_url = model_url.replace("model.onnx", "tokenizer.json");

        // Download model if not exists
        if !model_path.exists() {
            self.download_file(model_url, &model_path).await?;
        }

        if !tokenizer_path.exists() {
            self.download_file(tokenizer_url, &tokenizer_path).await?;
        }

        // Load the model
        let generator = OnnxEmbeddingGenerator::new(model_path, tokenizer_path)?;
        let generator_arc = Arc::new(generator);

        self.models.insert(model_name.to_string(), Arc::clone(&generator_arc));
        Ok(generator_arc)
    }

    /// Download a file from URL
    async fn download_file(&self, url: &str, path: &PathBuf) -> Result<()> {
        let response = reqwest::get(url).await
            .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to download model: {}", e)))?;

        let bytes = response.bytes().await
            .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to read model data: {}", e)))?;

        std::fs::write(path, &bytes)
            .map_err(|e| CodeSearchError::EmbeddingError(format!("Failed to save model: {}", e)))?;

        Ok(())
    }
}

impl Default for ModelManager {
    fn default() -> Self {
        Self::new().unwrap()
    }
}