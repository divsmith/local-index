// ABOUTME: Rust language parser using Tree-sitter

use crate::error::{CodeSearchError, Result};
use crate::parsers::{LanguageParser, ParseResult, Symbol, SymbolKind};
use tree_sitter_rust::language;
use tree_sitter::{Language, Node, Parser, Tree};

pub struct RustParser;

impl LanguageParser for RustParser {
    fn language(&self) -> Language {
        language()
    }

    fn file_extensions(&self) -> Vec<&'static str> {
        vec!["rs"]
    }

    fn parse_file(&self, content: &str) -> Result<ParseResult> {
        let mut parser = Parser::new();
        parser.set_language(self.language())
            .map_err(|_| CodeSearchError::Parse("Failed to set Rust language".to_string()))?;

        let tree = parser.parse(content, None)
            .ok_or(CodeSearchError::Parse("Failed to parse Rust code".to_string()))?;

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

impl RustParser {
    fn extract_symbols_recursive(
        &self,
        node: &Node,
        content: &str,
        symbols: &mut Vec<Symbol>,
    ) {
        match node.kind() {
            "function_item" => {
                self.extract_function(node, content, symbols);
            }
            "struct_item" => {
                self.extract_struct(node, content, symbols);
            }
            "enum_item" => {
                self.extract_enum(node, content, symbols);
            }
            "trait_item" => {
                self.extract_trait(node, content, symbols);
            }
            "impl_item" => {
                self.extract_impl(node, content, symbols);
            }
            "mod_item" => {
                self.extract_module(node, content, symbols);
            }
            "use_declaration" => {
                self.extract_import(node, content, symbols);
            }
            "const_item" => {
                self.extract_const(node, content, symbols);
            }
            "static_item" => {
                self.extract_static(node, content, symbols);
            }
            "type_alias_declaration" => {
                self.extract_type_alias(node, content, symbols);
            }
            "macro_definition" => {
                self.extract_macro(node, content, symbols);
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

    fn extract_struct(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Struct,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_enum(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Enum,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_trait(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Trait,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_impl(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(type_node) = node.child_by_field_name("type") {
            let type_name = content[type_node.start_byte()..type_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                type_name,
                SymbolKind::Impl,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_module(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Module,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_import(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(argument_node) = node.child_by_field_name("argument") {
            let import_path = content[argument_node.start_byte()..argument_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                import_path,
                SymbolKind::Import,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_const(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Const,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_static(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Static,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_type_alias(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::TypeAlias,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }

    fn extract_macro(&self, node: &Node, content: &str, symbols: &mut Vec<Symbol>) {
        if let Some(name_node) = node.child_by_field_name("name") {
            let name = content[name_node.start_byte()..name_node.end_byte()].to_string();
            symbols.push(Symbol::new(
                name,
                SymbolKind::Macro,
                node.start_position().row + 1,
                node.end_position().row + 1,
                node.start_byte(),
                node.end_byte(),
            ));
        }
    }
}