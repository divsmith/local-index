// ABOUTME: Integration tests for codesearch

use codesearch::*;
use std::path::PathBuf;
use tempfile::TempDir;

#[tokio::test]
async fn test_full_workflow() -> Result<(), Box<dyn std::error::Error>> {
    // Create temporary directory with test code
    let temp_dir = TempDir::new()?;
    create_test_code_files(temp_dir.path())?;

    // Initialize index manager
    let index_manager = codesearch::storage::IndexManager::new(temp_dir.path())?;

    // Index the directory
    let progress = index_manager.index_directory()?;
    assert!(progress.processed_files > 0, "Should have processed at least one file");
    assert!(progress.completion_rate() > 0.8, "Should have high completion rate");

    // Initialize search engine
    let mut search_engine = codesearch::search::SearchEngine::new(temp_dir.path())?;

    // Test semantic search
    let search_query = codesearch::search::SearchQuery {
        text: "fibonacci function".to_string(),
        query_type: codesearch::search::QueryType::Semantic,
        filters: codesearch::search::SearchFilters::default(),
        limit: 10,
    };

    let results = search_engine.search(&search_query).await?;
    assert!(!results.is_empty(), "Should find search results for fibonacci");

    // Test symbol search
    let symbol_query = codesearch::search::SearchQuery {
        text: "fibonacci".to_string(),
        query_type: codesearch::search::QueryType::Symbol,
        filters: codesearch::search::SearchFilters::default(),
        limit: 10,
    };

    let symbol_results = search_engine.search(&symbol_query).await?;
    assert!(!symbol_results.is_empty(), "Should find fibonacci symbol");

    Ok(())
}

#[tokio::test]
async fn test_search_engine_basic_functionality() -> Result<(), Box<dyn std::error::Error>> {
    let temp_dir = TempDir::new()?;
    create_simple_test_files(temp_dir.path())?;

    // Index first
    let index_manager = codesearch::storage::IndexManager::new(temp_dir.path())?;
    let progress = index_manager.index_directory()?;
    assert!(progress.is_complete(), "Indexing should complete");

    // Test search engine
    let mut search_engine = codesearch::search::SearchEngine::new(temp_dir.path())?;
    assert!(search_engine.is_indexed(), "Project should be indexed");

    // Test search with different query types
    let mut query = codesearch::search::SearchQuery::default();
    query.text = "test function".to_string();

    // Semantic search
    query.query_type = codesearch::search::QueryType::Semantic;
    let results = search_engine.search(&query).await?;

    // Symbol search
    query.query_type = codesearch::search::QueryType::Symbol;
    let symbol_results = search_engine.search(&query).await?;

    // At least one type should find results
    assert!(results.len() > 0 || symbol_results.len() > 0,
            "Should find results with either semantic or symbol search");

    Ok(())
}

#[test]
fn test_parser_registry() -> Result<(), Box<dyn std::error::Error>> {
    let registry = codesearch::parsers::ParserRegistry::new();

    // Test supported extensions
    let extensions = registry.get_supported_extensions();
    assert!(extensions.contains(&"rs".to_string()), "Should support Rust files");
    assert!(extensions.contains(&"py".to_string()), "Should support Python files");
    assert!(extensions.contains(&"md".to_string()), "Should support Markdown files");

    // Test file detection
    let rust_file = PathBuf::from("test.rs");
    let python_file = PathBuf::from("test.py");
    let markdown_file = PathBuf::from("test.md");
    let unsupported_file = PathBuf::from("test.xyz");

    assert!(registry.is_file_supported(&rust_file), "Should support Rust files");
    assert!(registry.is_file_supported(&python_file), "Should support Python files");
    assert!(registry.is_file_supported(&markdown_file), "Should support Markdown files");
    assert!(!registry.is_file_supported(&unsupported_file), "Should not support unsupported files");

    Ok(())
}

#[test]
fn test_file_scanner() -> Result<(), Box<dyn std::error::Error>> {
    let temp_dir = TempDir::new()?;
    create_test_directory_structure(temp_dir.path())?;

    let scanner = codesearch::filesystem::FileScanner::new();
    let files = scanner.scan_directory(temp_dir.path())?;

    assert!(!files.is_empty(), "Should find some files to scan");

    // Should not include excluded directories
    let found_git = files.iter().any(|f| f.to_string_lossy().contains(".git"));
    assert!(!found_git, "Should not include .git directory");

    // Should include supported file types
    let found_rust = files.iter().any(|f| f.extension().map_or(false, |ext| ext == "rs"));
    let found_python = files.iter().any(|f| f.extension().map_or(false, |ext| ext == "py"));
    assert!(found_rust || found_python, "Should include supported file types");

    Ok(())
}

fn create_test_code_files(dir: &std::path::Path) -> Result<(), std::io::Error> {
    use std::fs;

    // Create test Python file with fibonacci
    fs::write(dir.join("math.py"), r#"
def fibonacci(n):
    """Calculate the nth Fibonacci number."""
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

def factorial(n):
    """Calculate factorial of n."""
    if n <= 1:
        return 1
    return n * factorial(n-1)

class Calculator:
    def __init__(self):
        self.history = []

    def add(self, a, b):
        result = a + b
        self.history.append(f"{a} + {b} = {result}")
        return result
"#)?;

    // Create test Rust file
    fs::write(dir.join("utils.rs"), r#"
pub fn fibonacci(n: u32) -> u32 {
    match n {
        0 => 0,
        1 => 1,
        n => fibonacci(n - 1) + fibonacci(n - 2),
    }
}

pub struct Calculator {
    history: Vec<String>,
}

impl Calculator {
    pub fn new() -> Self {
        Self {
            history: Vec::new(),
        }
    }

    pub fn add(&mut self, a: i32, b: i32) -> i32 {
        let result = a + b;
        self.history.push(format!("{} + {} = {}", a, b, result));
        result
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_fibonacci() {
        assert_eq!(fibonacci(0), 0);
        assert_eq!(fibonacci(1), 1);
        assert_eq!(fibonacci(5), 5);
    }

    #[test]
    fn test_calculator() {
        let mut calc = Calculator::new();
        assert_eq!(calc.add(2, 3), 5);
    }
}
"#)?;

    // Create README
    fs::write(dir.join("README.md"), r#"
# Test Project

This is a test project for codesearch integration testing.

## Features

- Fibonacci implementation in both Python and Rust
- Calculator class/struct
- Comprehensive test coverage

## Usage

```python
from math import fibonacci
print(fibonacci(10))
```

```rust
use utils::fibonacci;
println!("{}", fibonacci(10));
```
"#)?;

    Ok(())
}

fn create_simple_test_files(dir: &std::path::Path) -> Result<(), std::io::Error> {
    use std::fs;

    fs::write(dir.join("simple.py"), r#"
def test_function():
    """A simple test function."""
    return "Hello, World!"

class TestClass:
    def method_one(self):
        return "Method one"

    def method_two(self):
        return "Method two"
"#)?;

    Ok(())
}

fn create_test_directory_structure(dir: &std::path::Path) -> Result<(), std::io::Error> {
    use std::fs;

    // Create main files
    fs::create_dir_all(dir.join("src"))?;
    fs::write(dir.join("src/main.rs"), "fn main() {\n    println!(\"Hello\");\n}\n")?;

    fs::write(dir.join("lib.py"), "def hello():\n    return \"Hello\"\n")?;

    // Create excluded directories
    fs::create_dir_all(dir.join(".git"))?;
    fs::write(dir.join(".git/config"), "[core]\n")?;

    fs::create_dir_all(dir.join("target"))?;
    fs::write(dir.join("target/debug"), "binary")?;

    fs::create_dir_all(dir.join("__pycache__"))?;
    fs::write(dir.join("__pycache__/test.pyc"), "compiled")?;

    Ok(())
}