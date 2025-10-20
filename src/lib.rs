// ABOUTME: Agent-first semantic code search tool library

//! # CodeSearch Library
//!
//! This library provides semantic and symbol-aware code search capabilities
//! for AI coding agents. It offers local-first search with vector embeddings
//! and AST-based symbol extraction.

pub mod cli;
pub mod config;
pub mod filesystem;
pub mod index;
pub mod models;
pub mod parsers;
pub mod search;
pub mod storage;
pub mod utils;
pub mod validation;

// Re-export main types for convenience
pub use cli::Cli;
pub use config::Config;
pub use error::{CodeSearchError, Result};

// Common types that will be used across modules
#[derive(Debug, Clone)]
pub struct SearchResult {
    pub file_path: std::path::PathBuf,
    pub start_line: usize,
    pub end_line: usize,
    pub score: f32,
    pub context: String,
    pub code_snippet: String,
}

pub mod error {
    use thiserror::Error;

    #[derive(Error, Debug)]
    pub enum CodeSearchError {
        #[error("IO error: {0}")]
        Io(#[from] std::io::Error),

        #[error("Database error: {0}")]
        Database(#[from] rusqlite::Error),

        #[error("Model error: {0}")]
        Model(String),

        #[error("Parse error: {0}")]
        Parse(String),

        #[error("Configuration error: {0}")]
        Config(String),

        #[error("Search error: {0}")]
        Search(String),

        #[error("File system error: {0}")]
        FileSystem(String),

        #[error("Storage error: {0}")]
        Storage(String),

        #[error("JSON error: {0}")]
        Json(#[from] serde_json::Error),
    }

    impl From<notify::Error> for CodeSearchError {
        fn from(err: notify::Error) -> Self {
            CodeSearchError::FileSystem(err.to_string())
        }
    }

    pub type Result<T> = std::result::Result<T, CodeSearchError>;
}