// ABOUTME: Language parsers for codesearch

use crate::error::Result;
use tree_sitter::{Language, Tree};

pub mod registry;
pub mod rust;
pub mod python;
pub mod markdown;

pub use registry::ParserRegistry;
pub use rust::RustParser;
pub use python::PythonParser;
pub use markdown::MarkdownParser;

pub trait LanguageParser: Send + Sync {
    fn language(&self) -> Language;
    fn file_extensions(&self) -> Vec<&'static str>;
    fn parse_file(&self, content: &str) -> Result<ParseResult>;
    fn extract_symbols(&self, tree: &Tree, content: &str) -> Vec<Symbol>;
}

#[derive(Debug)]
pub struct ParseResult {
    pub tree: Tree,
    pub symbols: Vec<Symbol>,
    pub embeddings: Vec<Vec<f32>>, // Will be populated in Task 1.4
}

#[derive(Debug, Clone)]
pub struct Symbol {
    pub name: String,
    pub kind: SymbolKind,
    pub start_line: usize,
    pub end_line: usize,
    pub start_byte: usize,
    pub end_byte: usize,
    pub parent: Option<String>,
}

#[derive(Debug, Clone)]
pub enum SymbolKind {
    Function,
    Class,
    Variable,
    Import,
    Module,
    Struct,
    Enum,
    Trait,
    Impl,
    Const,
    Static,
    TypeAlias,
    Macro,
    // Add more as needed
}

impl Symbol {
    pub fn new(
        name: String,
        kind: SymbolKind,
        start_line: usize,
        end_line: usize,
        start_byte: usize,
        end_byte: usize,
    ) -> Self {
        Self {
            name,
            kind,
            start_line,
            end_line,
            start_byte,
            end_byte,
            parent: None,
        }
    }

    pub fn with_parent(mut self, parent: String) -> Self {
        self.parent = Some(parent);
        self
    }
}