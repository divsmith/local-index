// ABOUTME: Markdown language parser using Tree-sitter

use crate::error::{CodeSearchError, Result};
use crate::parsers::{LanguageParser, ParseResult, Symbol, SymbolKind};
use tree_sitter_md::language;
use tree_sitter::{Language, Node, Parser, Tree};

pub struct MarkdownParser;

impl LanguageParser for MarkdownParser {
    fn language(&self) -> Language {
        language()
    }

    fn file_extensions(&self) -> Vec<&'static str> {
        vec!["md", "markdown", "mdown", "mkdn"]
    }

    fn parse_file(&self, content: &str) -> Result<ParseResult> {
        let mut parser = Parser::new();
        parser.set_language(self.language())
            .map_err(|_| CodeSearchError::Parse("Failed to set Markdown language".to_string()))?;

        let tree = parser.parse(content, None)
            .ok_or(CodeSearchError::Parse("Failed to parse Markdown content".to_string()))?;

        let symbols = self.extract_symbols(&tree, content);

        Ok(ParseResult {
            tree,
            symbols,
            embeddings: Vec::new(), // Will be added in Task 1.4
        })
    }

    fn extract_symbols(&self, tree: &Tree, content: &str) -> Vec<Symbol> {
        let mut symbols = Vec::new();
        self.extract_symbols_recursive(&tree.root_node(), content, &mut symbols);
        symbols
    }
}

impl MarkdownParser {
    fn extract_symbols_recursive(
        &self,
        node: &Node,
        content: &str,
        symbols: &mut Vec<Symbol>,
    ) {
        match node.kind() {
            "atx_heading" => {
                self.extract_heading(node, content, symbols);
            }
            "setext_heading" => {
                self.extract_heading(node, content, symbols);
            }
            "fenced_code_block" => {
                self.extract_code_block(node, content, symbols);
            }
            "indented_code_block" => {
                self.extract_code_block(node, content, symbols);
            }
            "link_reference_definition" => {
                self.extract_link_reference(node, content, symbols);
            }
            _ => {}
        }

        // Recursively visit child nodes
        let mut cursor = node.walk();
        for child in node.children(&mut cursor) {
            self.extract_symbols_recursive(&child, content, symbols);
        }
    }

    fn extract_heading(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        // Extract heading text for symbol identification
        let mut heading_text = String::new();
        let mut cursor = node.walk();

        for child in node.children(&mut cursor) {
            if child.kind() == "heading_content" {
                heading_text = content[child.start_byte()..child.end_byte()].to_string();
                break;
            }
        }

        if !heading_text.is_empty() {
            // Determine heading level (1-6)
            let level = if node.kind() == "atx_heading" {
                // Count # characters for ATX headings
                content[node.start_byte()..node.start_byte() + 6]
                    .chars()
                    .take_while(|c| *c == '#')
                    .count()
            } else {
                // For setext headings, we'll default to level 1
                1
            };

            let symbol_name = if level > 1 {
                format!("H{}: {}", level, heading_text)
            } else {
                heading_text
            };

            symbols.push(Symbol::new(
                symbol_name,
                SymbolKind::Module, // Use Module kind for headings
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_code_block(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        // Extract code blocks as potential symbols
        let mut code_info = String::new();
        let mut cursor = node.walk();

        if node.kind() == "fenced_code_block" {
            for child in node.children(&mut cursor) {
                if child.kind() == "info_string" {
                    code_info = content[child.start_byte()..child.end_byte()].to_string();
                    break;
                }
            }
        }

        let symbol_name = if code_info.is_empty() {
            "Code Block".to_string()
        } else {
            format!("Code Block ({})", code_info)
        };

        symbols.push(Symbol::new(
            symbol_name,
            SymbolKind::Module, // Use Module kind for code blocks
            node.start_position().row + 1,
            node.end_position().row + 1,
            node.start_byte(),
            node.end_byte(),
        ));
    }

    fn extract_link_reference(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        // Extract link reference definitions
        if let Some(label_node) = node.child_by_field_name("label") {
            let label = content[label_node.start_byte()..label_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                format!("[{}]", label),
                SymbolKind::Import, // Use Import kind for link references
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }
}