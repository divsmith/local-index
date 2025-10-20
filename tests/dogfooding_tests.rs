// ABOUTME: Dogfooding tests - tool searching its own codebase

use codesearch::*;
use std::path::Path;

#[test]
fn test_parser_handles_own_source() -> Result<(), Box<dyn std::error::Error>> {
    let registry = codesearch::parsers::ParserRegistry::new();

    // Test that the tool can parse its own source files
    let project_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    let src_dir = project_root.join("src");

    // Find some Rust source files
    let rust_files = std::fs::read_dir(&src_dir)?
        .filter_map(|entry| entry.ok())
        .filter(|entry| {
            entry.path().extension().map_or(false, |ext| ext == "rs")
        })
        .take(3) // Test a few files to avoid long test times
        .collect::<Vec<_>>();

    assert!(!rust_files.is_empty(), "Should find some Rust source files");

    for file_entry in rust_files {
        let file_path = file_entry.path();
        let content = std::fs::read_to_string(&file_path)?;

        // Try to parse the file
        if registry.is_file_supported(&file_path) {
            let parse_result = registry.parse_file(&file_path, &content)?;

            // Should successfully parse and extract symbols
            println!("Parsed {}: {} symbols extracted",
                     file_path.file_name().unwrap().to_string_lossy(),
                     parse_result.symbols.len());
        }
    }

    Ok(())
}

#[test]
fn test_file_scanner_on_own_codebase() -> Result<(), Box<dyn std::error::Error>> {
    let scanner = codesearch::filesystem::FileScanner::new();
    let project_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"));

    // Scan the project directory
    let files = scanner.scan_directory(&project_root)?;

    assert!(!files.is_empty(), "Should find files in own codebase");

    // Should find source files
    let src_files = files.iter()
        .filter(|f| f.to_string_lossy().contains("src/"))
        .count();
    assert!(src_files > 0, "Should find source files");

    // Should exclude build directories
    let target_files = files.iter()
        .filter(|f| f.to_string_lossy().contains("target/"))
        .count();
    assert_eq!(target_files, 0, "Should exclude target directory");

    println!("Found {} files in own codebase", files.len());
    println!("Source files: {}", src_files);

    Ok(())
}

#[test]
fn test_symbol_extraction_from_own_code() -> Result<(), Box<dyn std::error::Error>> {
    let registry = codesearch::parsers::ParserRegistry::new();
    let project_root = PathBuf::from(env!("CARGO_MANIFEST_DIR"));

    // Find a specific source file to test symbol extraction
    let test_files = [
        "src/lib.rs",
        "src/cli/commands.rs",
        "src/search/engine.rs",
    ];

    for relative_path in &test_files {
        let full_path = project_root.join(relative_path);
        if full_path.exists() {
            let content = std::fs::read_to_string(&full_path)?;

            if registry.is_file_supported(&full_path) {
                let symbols = registry.extract_symbols_from_content(&full_path, &content)?;

                println!("Symbols in {}:", relative_path);
                for symbol in &symbols {
                    println!("  {} ({:?}) lines {}-{}",
                             symbol.name, symbol.kind,
                             symbol.start_line, symbol.end_line);
                }

                // Should find at least some symbols in non-trivial files
                if relative_path != &"src/lib.rs" { // lib.rs might be mostly re-exports
                    assert!(!symbols.is_empty(),
                           "Should find symbols in {}", relative_path);
                }
            }
        }
    }

    Ok(())
}

#[test]
fn test_mock_embedding_generation() -> Result<(), Box<dyn std::error::Error>> {
    use codesearch::models::{ModelManager, ModelConfig, EmbeddingGenerator};

    // Test that mock embedding generation works
    let model_manager = ModelManager::new(ModelConfig::default())?;
    let embedding_generator = EmbeddingGenerator::new(model_manager);

    // Test query embedding generation
    let query_embedding = embedding_generator.generate_query_embedding("search engine")?;
    assert_eq!(query_embedding.len(), 768, "Should generate 768-dimensional embedding");

    // Test file embedding generation with some sample code
    let sample_code = r#"
pub struct SearchEngine {
    files: Vec<String>,
}

impl SearchEngine {
    pub fn new() -> Self {
        Self { files: Vec::new() }
    }
}
"#;

    let mock_symbols = vec![
        codesearch::parsers::Symbol::new(
            "SearchEngine".to_string(),
            codesearch::parsers::SymbolKind::Struct,
            1, 10, 0, 100
        ),
        codesearch::parsers::Symbol::new(
            "new".to_string(),
            codesearch::parsers::SymbolKind::Function,
            7, 9, 50, 90
        ),
    ];

    let embeddings = embedding_generator.generate_file_embeddings(sample_code, &mock_symbols)?;
    assert!(!embeddings.is_empty(), "Should generate embeddings for code chunks");

    println!("Generated {} embeddings for sample code", embeddings.len());
    for embedding in &embeddings {
        println!("  Chunk type: {:?}, embedding size: {}",
                 embedding.chunk.chunk_type, embedding.embedding.len());
    }

    Ok(())
}

#[test]
fn test_vector_storage_operations() -> Result<(), Box<dyn std::error::Error>> {
    use tempfile::NamedTempFile;
    use codesearch::storage::VectorStorage;

    let temp_file = NamedTempFile::new()?;
    let mut storage = VectorStorage::create(temp_file.path(), 768)?;

    // Create test vectors
    let test_vectors = vec![
        vec![0.1; 768],
        vec![0.5; 768],
        vec![0.9; 768],
    ];

    // Store vectors
    let offsets = storage.append_vectors(&test_vectors)?;
    assert_eq!(offsets, vec![0, 1, 2]);

    // Retrieve vectors
    let retrieved_vectors = storage.get_vectors(&offsets)?;
    assert_eq!(retrieved_vectors.len(), 3);

    for (original, retrieved) in test_vectors.iter().zip(retrieved_vectors.iter()) {
        for (orig_val, ret_val) in original.iter().zip(retrieved.iter()) {
            assert!((orig_val - ret_val).abs() < f32::EPSILON);
        }
    }

    println!("Successfully stored and retrieved {} vectors", test_vectors.len());
    Ok(())
}

#[test]
fn test_search_query_types() -> Result<(), Box<dyn std::error::Error>> {
    use codesearch::search::{SearchQuery, QueryType, SearchFilters};

    // Test different query types
    let semantic_query = SearchQuery {
        text: "search functionality".to_string(),
        query_type: QueryType::Semantic,
        filters: SearchFilters::default(),
        limit: 10,
    };

    let symbol_query = SearchQuery {
        text: "SearchEngine".to_string(),
        query_type: QueryType::Symbol,
        filters: SearchFilters::default(),
        limit: 5,
    };

    let hybrid_query = SearchQuery {
        text: "find search engine implementation".to_string(),
        query_type: QueryType::Hybrid,
        filters: SearchFilters {
            file_types: vec!["rs".to_string()],
            min_score: 0.5,
            ..Default::default()
        },
        limit: 20,
    };

    // Test that queries are constructed correctly
    assert!(matches!(semantic_query.query_type, QueryType::Semantic));
    assert!(matches!(symbol_query.query_type, QueryType::Symbol));
    assert!(matches!(hybrid_query.query_type, QueryType::Hybrid));
    assert_eq!(hybrid_query.filters.file_types.len(), 1);
    assert_eq!(hybrid_query.filters.min_score, 0.5);

    println!("All query types constructed successfully");
    Ok(())
}