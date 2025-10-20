// ABOUTME: CLI entry point for agent-first semantic code search tool

use codesearch::{cli, error::Result};
use clap::Parser;
use tracing_subscriber;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize logging
    tracing_subscriber::fmt::init();

    // Parse CLI arguments
    let cli = cli::Cli::parse();

    // Route to appropriate command handler
    cli::handle_command(cli).await
}