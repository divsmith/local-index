// ABOUTME: Configuration management for codesearch

use crate::error::Result;
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Config {
    pub indexing: IndexingConfig,
    pub search: SearchConfig,
    pub models: ModelConfig,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IndexingConfig {
    pub max_file_size: u64,
    pub exclude_patterns: Vec<String>,
    pub supported_extensions: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SearchConfig {
    pub default_limit: usize,
    pub semantic_threshold: f32,
    pub symbol_weight: f32,
    pub semantic_weight: f32,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ModelConfig {
    pub default_model: String,
    pub memory_limit_mb: usize,
}

impl Default for Config {
    fn default() -> Self {
        Self {
            indexing: IndexingConfig {
                max_file_size: 10 * 1024 * 1024, // 10MB
                exclude_patterns: vec![
                    "node_modules".to_string(),
                    ".git".to_string(),
                    "target".to_string(),
                    "__pycache__".to_string(),
                ],
                supported_extensions: vec![
                    "rs".to_string(),
                    "py".to_string(),
                    "js".to_string(),
                    "ts".to_string(),
                    "yaml".to_string(),
                    "yml".to_string(),
                    "json".to_string(),
                    "md".to_string(),
                ],
            },
            search: SearchConfig {
                default_limit: 20,
                semantic_threshold: 0.7,
                symbol_weight: 0.6,
                semantic_weight: 0.4,
            },
            models: ModelConfig {
                default_model: "codebert-small".to_string(),
                memory_limit_mb: 1024,
            },
        }
    }
}

pub fn load_config() -> Result<Config> {
    // For now, return default config
    // Will implement file loading in later phases
    Ok(Config::default())
}