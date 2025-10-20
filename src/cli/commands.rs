// ABOUTME: CLI command definitions and implementations

use clap::{Parser, Subcommand, ValueEnum};
use crate::error::Result;
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "codesearch")]
#[command(about = "Agent-first semantic code search tool")]
#[command(version = "0.1.0")]
#[command(after_help = "
Examples:
  codesearch index .                    # Index current directory
  codesearch search 'auth' --type py   # Search Python files
  codesearch find 'main' --exact       # Find exact symbol matches
  codesearch validate --performance    # Run performance validation

For more help on a specific command:
  codesearch <command> --help
")]
pub struct Cli {
    #[command(subcommand)]
    pub command: Commands,

    /// Output results in JSON format (useful for scripts and agents)
    #[arg(short, long, global = true)]
    pub json: bool,

    /// Quiet mode - suppress non-error output
    #[arg(short, long, global = true)]
    pub quiet: bool,

    /// Maximum number of results to return
    #[arg(short, long, global = true, default_value = "20")]
    pub limit: usize,

    /// Verbose output (-v for normal verbose, -vv for very verbose)
    #[arg(short, long, global = true, action = clap::ArgAction::Count)]
    pub verbose: u8,
}

#[derive(Subcommand)]
pub enum Commands {
    /// Initialize and index a directory
    Index {
        /// Path to index (default: current directory)
        #[arg(default_value = ".")]
        path: PathBuf,

        /// Force full reindex instead of incremental
        #[arg(long)]
        force: bool,
    },

    /// Search for code using semantic queries
    Search {
        /// Search query
        query: String,

        /// Filter by file type
        #[arg(short = 't', long)]
        r#type: Option<String>,
    },

    /// Find specific symbols, functions, or classes
    Find {
        /// Symbol name to search for
        symbol: String,

        /// Search for exact matches only
        #[arg(long)]
        exact: bool,
    },

    /// Show indexing status and statistics
    Status {
        /// Path to check (default: current directory)
        #[arg(default_value = ".")]
        path: PathBuf,
    },

    /// Run validation tests for performance and platform compatibility
    Validate {
        /// Type of validation to run
        #[arg(long, default_value = "all")]
        validation_type: ValidationType,
    },
}

#[derive(ValueEnum, Clone)]
pub enum ValidationType {
    All,
    Performance,
    Platform,
}

// Command handlers
pub async fn handle_index(
    path: PathBuf,
    force: bool,
    json: bool,
    quiet: bool,
    verbose: u8,
) -> Result<()> {
    use crate::storage::IndexManager;

    if verbose > 0 && !quiet {
        eprintln!("Starting indexing for: {:?}", path);
    }

    let index_manager = IndexManager::new(&path)?;

    let progress = if force {
        index_manager.rebuild_index()?
    } else {
        index_manager.incremental_index()?
    };

    if !quiet {
        eprintln!("Indexing complete: {}/{} files processed",
                 progress.processed_files, progress.total_files);

        if progress.has_errors() {
            eprintln!("Encountered {} errors during indexing:", progress.errors.len());
            for error in &progress.errors[..progress.errors.len().min(5)] {
                eprintln!("  - {}: {}", error.file_path.display(), error.error);
            }
            if progress.errors.len() > 5 {
                eprintln!("  ... and {} more errors", progress.errors.len() - 5);
            }
        }
    }

    if json {
        let output = serde_json::json!({
            "total_files": progress.total_files,
            "processed_files": progress.processed_files,
            "errors": progress.errors.len(),
            "completion_rate": progress.completion_rate(),
            "success": progress.is_complete() && !progress.has_errors()
        });
        println!("{}", serde_json::to_string_pretty(&output)?);
    }

    Ok(())
}

pub async fn handle_search(
    query: String,
    file_type: Option<String>,
    limit: usize,
    json: bool,
    quiet: bool,
    verbose: u8,
) -> Result<()> {
    use crate::search::{SearchEngine, SearchQuery, QueryType, SearchFilters};

    if verbose > 0 && !quiet {
        eprintln!("Searching for: {}", query);
    }

    let mut search_engine = SearchEngine::new(".")?;

    if !search_engine.is_indexed() {
        if !quiet {
            eprintln!("No index found. Please run 'codesearch index .' first.");
        }
        return Ok(());
    }

    let mut filters = SearchFilters::default();
    if let Some(ft) = file_type {
        filters.file_types.push(ft);
    }

    let search_query = SearchQuery {
        text: query.clone(),
        query_type: QueryType::Hybrid,
        filters,
        limit,
    };

    let results = search_engine.search(&search_query).await?;

    if json {
        let json_results: Vec<serde_json::Value> = results.iter().map(|r| {
            serde_json::json!({
                "file_path": r.file_path,
                "start_line": r.start_line,
                "end_line": r.end_line,
                "score": r.score,
                "result_type": format!("{:?}", r.result_type),
                "context": r.context,
                "code_snippet": r.code_snippet,
                "symbols": r.symbols,
                "chunk_type": r.chunk_type
            })
        }).collect();

        let output = serde_json::json!({
            "query": query,
            "total_results": results.len(),
            "results": json_results
        });
        println!("{}", serde_json::to_string_pretty(&output)?);
    } else {
        if results.is_empty() && !quiet {
            eprintln!("No results found for query: {}", query);
        }

        for (i, result) in results.iter().enumerate() {
            if i > 0 { println!(); }

            println!("Result {} (score: {:.3})", i + 1, result.score);
            println!("File: {}:{}-{}",
                     result.file_path.display(),
                     result.start_line,
                     result.end_line);
            println!("Type: {}", result.chunk_type);
            if !result.symbols.is_empty() {
                println!("Symbols: {}", result.symbols.join(", "));
            }
            println!();
            println!("{}", result.code_snippet);
        }
    }

    Ok(())
}

pub async fn handle_find(
    symbol: String,
    exact: bool,
    limit: usize,
    json: bool,
    quiet: bool,
    verbose: u8,
) -> Result<()> {
    use crate::search::{SearchEngine, SearchQuery, QueryType, SearchFilters};

    if verbose > 0 && !quiet {
        eprintln!("Finding symbol: {}", symbol);
    }

    let mut search_engine = SearchEngine::new(".")?;

    if !search_engine.is_indexed() {
        if !quiet {
            eprintln!("No index found. Please run 'codesearch index .' first.");
        }
        return Ok(());
    }

    let mut filters = SearchFilters::default();
    if exact {
        filters.min_score = 1.0; // Only exact matches
    } else {
        filters.min_score = 0.3; // Allow fuzzy matches
    }

    let search_query = SearchQuery {
        text: symbol.clone(),
        query_type: QueryType::Symbol,
        filters,
        limit,
    };

    let results = search_engine.search(&search_query).await?;

    if json {
        let json_results: Vec<serde_json::Value> = results.iter().map(|r| {
            serde_json::json!({
                "file_path": r.file_path,
                "start_line": r.start_line,
                "end_line": r.end_line,
                "score": r.score,
                "result_type": format!("{:?}", r.result_type),
                "code_snippet": r.code_snippet,
                "symbols": r.symbols
            })
        }).collect();

        let output = serde_json::json!({
            "symbol": symbol,
            "exact": exact,
            "total_results": results.len(),
            "results": json_results
        });
        println!("{}", serde_json::to_string_pretty(&output)?);
    } else {
        if results.is_empty() && !quiet {
            eprintln!("No matches found for symbol: {}", symbol);
        }

        for (i, result) in results.iter().enumerate() {
            if i > 0 { println!(); }

            let match_type = match result.result_type {
                crate::search::SearchResultType::ExactSymbolMatch => "Exact",
                crate::search::SearchResultType::FuzzySymbolMatch(_) => "Fuzzy",
                _ => "Other",
            };

            println!("{} match (score: {:.3})", match_type, result.score);
            println!("File: {}:{}",
                     result.file_path.display(),
                     result.start_line);
            if !result.symbols.is_empty() {
                println!("Symbol: {}", result.symbols[0]);
            }
            println!();
            println!("{}", result.code_snippet);
        }
    }

    Ok(())
}

pub async fn handle_status(
    path: PathBuf,
    json: bool,
    quiet: bool,
    verbose: u8,
) -> Result<()> {
    use crate::search::SearchEngine;
    use crate::storage::IndexManager;

    if verbose > 0 && !quiet {
        eprintln!("Checking status for: {:?}", path);
    }

    let search_engine = SearchEngine::new(&path)?;
    let is_indexed = search_engine.is_indexed();

    if !is_indexed {
        if !quiet {
            eprintln!("No index found for: {:?}", path);
            eprintln!("Run 'codesearch index {:?}' to create an index.", path);
        }

        if json {
            let output = serde_json::json!({
                "path": path,
                "indexed": false,
                "message": "No index found"
            });
            println!("{}", serde_json::to_string_pretty(&output)?);
        }

        return Ok(());
    }

    let index_manager = IndexManager::new(&path)?;
    let stats = index_manager.get_statistics()?;

    if json {
        let output = serde_json::json!({
            "path": path,
            "indexed": true,
            "statistics": {
                "total_files": stats.total_files,
                "total_symbols": stats.total_symbols,
                "total_chunks": stats.total_chunks,
                "total_size_bytes": stats.total_size,
                "last_indexed": stats.last_indexed
            }
        });
        println!("{}", serde_json::to_string_pretty(&output)?);
    } else {
        println!("Index status for: {:?}", path);
        println!("  Files indexed: {}", stats.total_files);
        println!("  Symbols found: {}", stats.total_symbols);
        println!("  Code chunks: {}", stats.total_chunks);
        println!("  Total size: {} bytes", stats.total_size);
        println!("  Last indexed: {:?}", stats.last_indexed);
    }

    Ok(())
}

pub async fn handle_validate(
    validation_type: ValidationType,
    _json: bool,
    quiet: bool,
    _verbose: u8,
) -> Result<()> {
    use crate::cli::validation;

    match validation_type {
        ValidationType::All => {
            if !quiet {
                eprintln!("Running all validation tests...");
            }
            validation::run_performance_validation()?;
            validation::run_platform_validation()?;
        }
        ValidationType::Performance => {
            if !quiet {
                eprintln!("Running performance validation tests...");
            }
            validation::run_performance_validation()?;
        }
        ValidationType::Platform => {
            if !quiet {
                eprintln!("Running platform compatibility tests...");
            }
            validation::run_platform_validation()?;
        }
    }

    Ok(())
}