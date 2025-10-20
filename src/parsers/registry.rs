// ABOUTME: Parser registry for managing language parsers

use crate::error::{CodeSearchError, Result};
use crate::parsers::{LanguageParser, ParseResult, RustParser, PythonParser, MarkdownParser};
use std::collections::HashMap;
use std::path::Path;

pub struct ParserRegistry {
    parsers: HashMap<String, Box<dyn LanguageParser>>,
    extension_map: HashMap<String, String>,
}

impl ParserRegistry {
    pub fn new() -> Self {
        let mut registry = Self {
            parsers: HashMap::new(),
            extension_map: HashMap::new(),
        };

        // Register built-in parsers
        registry.register_parser("rust", Box::new(RustParser));
        registry.register_parser("python", Box::new(PythonParser));
        registry.register_parser("markdown", Box::new(MarkdownParser));

        registry
    }

    pub fn register_parser(&mut self, name: &str, parser: Box<dyn LanguageParser>) {
        // Register parser by name
        self.parsers.insert(name.to_string(), parser);

        // Update extension map
        if let Some(parser_ref) = self.parsers.get(name) {
            for extension in parser_ref.file_extensions() {
                self.extension_map.insert(extension.to_string(), name.to_string());
            }
        }
    }

    pub fn get_parser_for_file(&self, file_path: &Path) -> Option<&dyn LanguageParser> {
        if let Some(extension) = file_path.extension().and_then(|ext| ext.to_str()) {
            if let Some(language_name) = self.extension_map.get(extension) {
                return self.parsers.get(language_name).map(|p| p.as_ref());
            }
        }
        None
    }

    pub fn parse_file(&self, file_path: &Path, content: &str) -> Result<ParseResult> {
        let parser = self.get_parser_for_file(file_path)
            .ok_or_else(|| CodeSearchError::Parse(format!(
                "No parser available for file: {:?}",
                file_path
            )))?;

        parser.parse_file(content)
    }

    pub fn extract_symbols_from_content(&self, file_path: &Path, content: &str) -> Result<Vec<crate::parsers::Symbol>> {
        let parse_result = self.parse_file(file_path, content)?;
        Ok(parse_result.symbols)
    }

    pub fn get_supported_extensions(&self) -> Vec<String> {
        self.extension_map.keys().cloned().collect()
    }

    pub fn get_supported_languages(&self) -> Vec<String> {
        self.parsers.keys().cloned().collect()
    }

    pub fn is_file_supported(&self, file_path: &Path) -> bool {
        self.get_parser_for_file(file_path).is_some()
    }
}

impl Default for ParserRegistry {
    fn default() -> Self {
        Self::new()
    }
}

