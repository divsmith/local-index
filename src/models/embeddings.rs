// ABOUTME: Vector embedding generation for code snippets

use crate::error::{CodeSearchError, Result};
use crate::models::ModelManager;
use crate::parsers::Symbol;
use chrono::Utc;
use std::path::Path;

#[derive(Debug, Clone)]
pub struct CodeChunk {
    pub start_line: usize,
    pub end_line: usize,
    pub start_byte: usize,
    pub end_byte: usize,
    pub chunk_type: ChunkType,
    pub symbol_name: Option<String>,
}

#[derive(Debug, Clone)]
pub enum ChunkType {
    Function,
    Class,
    Struct,
    Module,
    Import,
    Variable,
    CodeBlock,
    Other,
}

impl CodeChunk {
    pub fn extract_text(&self, content: &str) -> String {
        content[self.start_byte..self.end_byte].to_string()
    }
}

pub struct EmbeddingGenerator {
    model_manager: ModelManager,
}

impl EmbeddingGenerator {
    pub fn new(model_manager: ModelManager) -> Self {
        Self { model_manager }
    }

    pub fn generate_file_embeddings(&self, file_content: &str, symbols: &[Symbol]) -> Result<Vec<FileEmbedding>> {
        // Split file into meaningful chunks based on symbols
        let chunks = self.chunk_code(file_content, symbols)?;

        // Generate texts for each chunk
        let chunk_texts: Vec<String> = chunks.iter()
            .map(|chunk| chunk.extract_text(file_content))
            .collect();

        // Generate embeddings for all chunks
        let embeddings = self.model_manager.generate_embeddings(&chunk_texts)?;

        // Combine chunk information with embeddings
        let mut file_embeddings = Vec::new();
        for (chunk, embedding) in chunks.into_iter().zip(embeddings.into_iter()) {
            file_embeddings.push(FileEmbedding {
                chunk,
                embedding,
                timestamp: Utc::now(),
            });
        }

        Ok(file_embeddings)
    }

    pub fn generate_query_embedding(&self, query: &str) -> Result<Vec<f32>> {
        let embeddings = self.model_manager.generate_embeddings(&[query.to_string()])?;
        Ok(embeddings.into_iter().next().unwrap_or_default())
    }

    fn chunk_code(&self, content: &str, symbols: &[Symbol]) -> Result<Vec<CodeChunk>> {
        let mut chunks = Vec::new();

        // Create chunks from symbols
        for symbol in symbols {
            let chunk_type = match symbol.kind {
                crate::parsers::SymbolKind::Function => ChunkType::Function,
                crate::parsers::SymbolKind::Class => ChunkType::Class,
                crate::parsers::SymbolKind::Struct => ChunkType::Struct,
                crate::parsers::SymbolKind::Module => ChunkType::Module,
                crate::parsers::SymbolKind::Import => ChunkType::Import,
                crate::parsers::SymbolKind::Variable => ChunkType::Variable,
                _ => ChunkType::Other,
            };

            chunks.push(CodeChunk {
                start_line: symbol.start_line,
                end_line: symbol.end_line,
                start_byte: symbol.start_byte,
                end_byte: symbol.end_byte,
                chunk_type,
                symbol_name: Some(symbol.name.clone()),
            });
        }

        // If no symbols were found, create a single chunk for the entire file
        if chunks.is_empty() {
            chunks.push(CodeChunk {
                start_line: 1,
                end_line: content.lines().count(),
                start_byte: 0,
                end_byte: content.len(),
                chunk_type: ChunkType::CodeBlock,
                symbol_name: None,
            });
        }

        Ok(chunks)
    }

    pub fn chunk_code_by_lines(&self, content: &str, chunk_size: usize) -> Result<Vec<CodeChunk>> {
        let lines: Vec<&str> = content.lines().collect();
        let mut chunks = Vec::new();

        for (start_idx, chunk_lines) in lines.chunks(chunk_size).enumerate() {
            let start_line = start_idx * chunk_size + 1;
            let end_line = start_line + chunk_lines.len() - 1;

            // Find byte positions
            let start_byte = content.lines().take(start_line - 1).map(|l| l.len() + 1).sum::<usize>();
            let end_byte = start_byte + chunk_lines.iter().map(|l| l.len() + 1).sum::<usize>();

            if start_byte < content.len() {
                chunks.push(CodeChunk {
                    start_line,
                    end_line,
                    start_byte,
                    end_byte: end_byte.min(content.len()),
                    chunk_type: ChunkType::CodeBlock,
                    symbol_name: None,
                });
            }
        }

        Ok(chunks)
    }
}

#[derive(Debug, Clone)]
pub struct FileEmbedding {
    pub chunk: CodeChunk,
    pub embedding: Vec<f32>,
    pub timestamp: chrono::DateTime<Utc>,
}

impl FileEmbedding {
    pub fn text(&self, content: &str) -> String {
        self.chunk.extract_text(content)
    }

    pub fn similarity(&self, other: &[f32]) -> f32 {
        cosine_similarity(&self.embedding, other)
    }
}

// Simple cosine similarity calculation
pub fn cosine_similarity(a: &[f32], b: &[f32]) -> f32 {
    if a.len() != b.len() {
        return 0.0;
    }

    let dot_product: f32 = a.iter().zip(b.iter()).map(|(x, y)| x * y).sum();
    let magnitude_a: f32 = a.iter().map(|x| x * x).sum::<f32>().sqrt();
    let magnitude_b: f32 = b.iter().map(|x| x * x).sum::<f32>().sqrt();

    if magnitude_a == 0.0 || magnitude_b == 0.0 {
        0.0
    } else {
        dot_product / (magnitude_a * magnitude_b)
    }
}