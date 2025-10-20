// ABOUTME: File discovery and scanning functionality

use crate::error::Result;
use std::path::{Path, PathBuf};
use walkdir::WalkDir;

pub struct FileScanner {
    pub exclude_patterns: Vec<String>,
    pub max_file_size: u64,
    pub supported_extensions: Vec<String>,
}

impl FileScanner {
    pub fn new() -> Self {
        Self {
            exclude_patterns: vec![
                "node_modules".to_string(),
                ".git".to_string(),
                "target".to_string(),
                "__pycache__".to_string(),
                ".vscode".to_string(),
                ".idea".to_string(),
            ],
            max_file_size: 10 * 1024 * 1024, // 10MB
            supported_extensions: vec![
                "rs".to_string(),
                "py".to_string(),
                "js".to_string(),
                "ts".to_string(),
                "yaml".to_string(),
                "yml".to_string(),
                "json".to_string(),
                "md".to_string(),
            ],
        }
    }

    pub fn scan_directory(&self, path: &Path) -> Result<Vec<PathBuf>> {
        let mut files = Vec::new();

        for entry in WalkDir::new(path)
            .follow_links(false)
            .into_iter()
            .filter_entry(|e| !self.is_excluded(e.path()))
        {
            let entry = entry.map_err(|e| crate::error::CodeSearchError::Io(e.into()))?;

            if entry.file_type().is_file() {
                if self.should_index_file(entry.path())? {
                    files.push(entry.path().to_path_buf());
                }
            }
        }

        Ok(files)
    }

    fn should_index_file(&self, path: &Path) -> Result<bool> {
        // Check file size
        let metadata = std::fs::metadata(path)?;
        if metadata.len() > self.max_file_size {
            return Ok(false);
        }

        // Check file extension
        if let Some(extension) = path.extension().and_then(|s| s.to_str()) {
            Ok(self.supported_extensions.contains(&extension.to_string()))
        } else {
            Ok(false)
        }
    }

    fn is_excluded(&self, path: &Path) -> bool {
        let path_str = path.to_string_lossy();
        self.exclude_patterns.iter()
            .any(|pattern| path_str.contains(pattern))
    }
}

impl Default for FileScanner {
    fn default() -> Self {
        Self::new()
    }
}

