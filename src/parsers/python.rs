// ABOUTME: Python language parser using Tree-sitter

use crate::error::{CodeSearchError, Result};
use crate::parsers::{LanguageParser, ParseResult, Symbol, SymbolKind};
use tree_sitter_python::language;
use tree_sitter::{Language, Node, Parser, Tree};

pub struct PythonParser;

impl LanguageParser for PythonParser {
    fn language(&self) -> Language {
        language()
    }

    fn file_extensions(&self) -> Vec<&'static str> {
        vec!["py", "pyw", "pyi"]
    }

    fn parse_file(&self, content: &str) -> Result<ParseResult> {
        let mut parser = Parser::new();
        parser.set_language(self.language())
            .map_err(|_| CodeSearchError::Parse("Failed to set Python language".to_string()))?;

        let tree = parser.parse(content, None)
            .ok_or(CodeSearchError::Parse("Failed to parse Python code".to_string()))?;

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

impl PythonParser {
    fn extract_symbols_recursive(
        &self,
        node: &Node,
        content: &str,
        symbols: &mut Vec<Symbol>,
    ) {
        match node.kind() {
            "function_definition" => {
                self.extract_function(node, content, symbols);
            }
            "class_definition" => {
                self.extract_class(node, content, symbols);
            }
            "assignment" => {
                self.extract_variable_assignment(node, content, symbols);
            }
            "import_statement" | "import_from_statement" => {
                self.extract_import(node, content, symbols);
            }
            "module" => {
                // Module is the root, we don't extract it as a symbol
            }
            _ => {}
        }

        // Recursively visit child nodes
        let mut cursor = node.walk();
        for child in node.children(&mut cursor) {
            self.extract_symbols_recursive(&child, content, symbols);
        }
    }

    fn extract_function(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Function,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_class(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Class,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_variable_assignment(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        // Only extract top-level variable assignments (module-level constants)
        if let Some(left_node) = node.child_by_field_name("left") {
            // Check if this is a simple assignment to an identifier
            if left_node.kind() == "identifier" {
                let name = content[left_node.start_byte()..left_node.end_byte()].to_string();
                symbols.push(Symbol::new(
                    name,
                    SymbolKind::Variable,
                    node.start_position().row + 1,
                    node.end_position().row + 1,
                    node.start_byte(),
                    node.end_byte(),
                ));
            }
        }
    }

    fn extract_import(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        match node.kind() {
            "import_statement" => {
                // Handle "import module" statements
                for child in node.children(&mut node.walk()) {
                    if child.kind() == "dotted_name" {
                        let import_path = content[child.start_byte()..child.end_byte()].to_string();
                        symbols.push(Symbol::new(
                            import_path,
                            SymbolKind::Import,
                            node.start_position().row + 1,
                            node.end_position().row + 1,
                            node.start_byte(),
                            node.end_byte(),
                        ));
                        break;
                    }
                }
            }
            "import_from_statement" => {
                // Handle "from module import name" statements
                if let Some(module_node) = node.child_by_field_name("module_name") {
                    let module_name = content[module_node.start_byte()..module_node.end_byte()].to_string();
                    symbols.push(Symbol::new(
                        module_name,
                        SymbolKind::Import,
                        node.start_position().row + 1,
                        node.end_position().row + 1,
                        node.start_byte(),
                        node.end_byte(),
                    ));
                }
            }
            _ => {}
        }
    }
}