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
    if !quiet {
        eprintln!("Index command not yet implemented");
        eprintln!("Path: {:?}", path);
        eprintln!("Force: {}", force);
        if verbose > 0 {
            eprintln!("JSON output: {}", json);
            eprintln!("Verbose level: {}", verbose);
        }
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
    if !quiet {
        eprintln!("Search command not yet implemented");
        eprintln!("Query: {}", query);
        if let Some(ft) = file_type {
            eprintln!("File type: {}", ft);
        }
        eprintln!("Limit: {}", limit);
        if verbose > 0 {
            eprintln!("JSON output: {}", json);
            eprintln!("Verbose level: {}", verbose);
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
    if !quiet {
        eprintln!("Find command not yet implemented");
        eprintln!("Symbol: {}", symbol);
        eprintln!("Exact: {}", exact);
        eprintln!("Limit: {}", limit);
        if verbose > 0 {
            eprintln!("JSON output: {}", json);
            eprintln!("Verbose level: {}", verbose);
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
    if !quiet {
        eprintln!("Status command not yet implemented");
        eprintln!("Path: {:?}", path);
        if verbose > 0 {
            eprintln!("JSON output: {}", json);
            eprintln!("Verbose level: {}", verbose);
        }
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