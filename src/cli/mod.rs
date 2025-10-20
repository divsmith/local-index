// ABOUTME: CLI command definitions and routing

pub mod commands;
pub mod validation;

use crate::error::Result;

pub use commands::{Cli, Commands};

pub async fn handle_command(cli: Cli) -> Result<()> {
    match cli.command {
        Commands::Index { path, force } => {
            commands::handle_index(path, force, cli.json, cli.quiet, cli.verbose).await
        }
        Commands::Search { query, r#type } => {
            commands::handle_search(query, r#type, cli.limit, cli.json, cli.quiet, cli.verbose).await
        }
        Commands::Find { symbol, exact } => {
            commands::handle_find(symbol, exact, cli.limit, cli.json, cli.quiet, cli.verbose).await
        }
        Commands::Status { path } => {
            commands::handle_status(path, cli.json, cli.quiet, cli.verbose).await
        }
        Commands::Validate { validation_type } => {
            commands::handle_validate(validation_type, cli.json, cli.quiet, cli.verbose).await
        }
    }
}