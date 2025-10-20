// ABOUTME: Path manipulation utilities

use std::path::{Path, PathBuf};

pub fn normalize_path(path: &Path) -> PathBuf {
    // Simple path normalization
    path.components().collect()
}

pub fn get_relative_path(base: &Path, target: &Path) -> Option<PathBuf> {
    pathdiff::diff_paths(target, base)
}

pub fn is_hidden(path: &Path) -> bool {
    path.file_name()
        .map(|name| name.to_string_lossy().starts_with('.'))
        .unwrap_or(false)
}

// Add a simple pathdiff implementation for testing
pub mod pathdiff {
    use std::path::{Path, PathBuf};

    pub fn diff_paths(_base: &Path, target: &Path) -> Option<PathBuf> {
        // Simplified implementation - just return target for now
        Some(target.to_path_buf())
    }
}