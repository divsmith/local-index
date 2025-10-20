// ABOUTME: File validation and metadata extraction

use crate::error::Result;
use std::fs;
use std::path::{Path, PathBuf};
use std::time::SystemTime;

#[derive(Debug, Clone)]
pub struct FileMetadata {
    pub path: PathBuf,
    pub size: u64,
    pub modified: SystemTime,
    pub created: Option<SystemTime>,
    pub is_binary: bool,
    pub language: Option<String>,
    pub encoding: Option<String>,
}

pub struct FileValidator {
    max_file_size: u64,
    binary_size_threshold: usize,
    supported_extensions: Vec<String>,
}

impl FileValidator {
    pub fn new() -> Self {
        Self {
            max_file_size: 10 * 1024 * 1024, // 10MB
            binary_size_threshold: 1024,    // 1KB for binary detection
            supported_extensions: vec![
                "rs".to_string(),
                "py".to_string(),
                "js".to_string(),
                "ts".to_string(),
                "jsx".to_string(),
                "tsx".to_string(),
                "yaml".to_string(),
                "yml".to_string(),
                "json".to_string(),
                "md".to_string(),
                "txt".to_string(),
                "toml".to_string(),
                "ini".to_string(),
                "cfg".to_string(),
                "sh".to_string(),
                "bash".to_string(),
                "zsh".to_string(),
                "fish".to_string(),
            ],
        }
    }

    pub fn validate_file(&self, path: &Path) -> Result<FileMetadata> {
        let metadata = fs::metadata(path)?;

        // Check file size
        if metadata.len() > self.max_file_size {
            return Err(crate::error::CodeSearchError::Search(format!(
                "File too large: {} bytes (max: {})",
                metadata.len(),
                self.max_file_size
            )));
        }

        // Check if it's a file
        if !metadata.is_file() {
            return Err(crate::error::CodeSearchError::Search(
                "Path is not a file".to_string()
            ));
        }

        // Determine if file is binary
        let is_binary = self.is_binary_file(path)?;

        // Detect language
        let language = self.detect_language(path);

        // Detect encoding (simplified)
        let encoding = if is_binary {
            None
        } else {
            Some("utf-8".to_string()) // Assume UTF-8 for text files
        };

        Ok(FileMetadata {
            path: path.to_path_buf(),
            size: metadata.len(),
            modified: metadata.modified()?,
            created: metadata.created().ok(),
            is_binary,
            language,
            encoding,
        })
    }

    pub fn is_binary_file(&self, path: &Path) -> Result<bool> {
        use std::io::Read;

        let mut file = fs::File::open(path)?;
        let mut buffer = [0; 512];

        let bytes_read = file.read(&mut buffer)?;

        if bytes_read == 0 {
            return Ok(false); // Empty file is not binary
        }

        // Check for null bytes (common in binary files)
        if buffer[..bytes_read].contains(&0) {
            return Ok(true);
        }

        // Check percentage of non-printable characters
        let non_printable = buffer[..bytes_read]
            .iter()
            .filter(|&&b| b < 32 && b != b'\n' && b != b'\r' && b != b'\t')
            .count();

        let non_printable_ratio = non_printable as f64 / bytes_read as f64;
        Ok(non_printable_ratio > 0.3) // If >30% non-printable, consider binary
    }

    pub fn detect_language(&self, path: &Path) -> Option<String> {
        if let Some(extension) = path.extension().and_then(|ext| ext.to_str()) {
            match extension.to_lowercase().as_str() {
                "rs" => Some("rust".to_string()),
                "py" => Some("python".to_string()),
                "js" => Some("javascript".to_string()),
                "jsx" => Some("javascript".to_string()),
                "ts" => Some("typescript".to_string()),
                "tsx" => Some("typescript".to_string()),
                "yaml" | "yml" => Some("yaml".to_string()),
                "json" => Some("json".to_string()),
                "md" => Some("markdown".to_string()),
                "txt" => Some("text".to_string()),
                "toml" => Some("toml".to_string()),
                "ini" | "cfg" => Some("config".to_string()),
                "sh" | "bash" | "zsh" | "fish" => Some("shell".to_string()),
                _ => None,
            }
        } else {
            // Check for special files without extensions
            if let Some(filename) = path.file_name().and_then(|name| name.to_str()) {
                match filename {
                    "Dockerfile" => Some("docker".to_string()),
                    "Makefile" => Some("makefile".to_string()),
                    "Rakefile" => Some("ruby".to_string()),
                    "Cargo.toml" => Some("toml".to_string()),
                    "package.json" => Some("json".to_string()),
                    _ => None,
                }
            } else {
                None
            }
        }
    }

    pub fn should_index_file(&self, metadata: &FileMetadata) -> bool {
        // Don't index binary files
        if metadata.is_binary {
            return false;
        }

        // Check if file has supported extension
        if let Some(language) = &metadata.language {
            self.supported_extensions.iter().any(|ext| {
                match language.as_str() {
                    "rust" => ext == "rs",
                    "python" => ext == "py",
                    "javascript" => ["js", "jsx"].contains(&ext.as_str()),
                    "typescript" => ["ts", "tsx"].contains(&ext.as_str()),
                    "yaml" => ["yaml", "yml"].contains(&ext.as_str()),
                    _ => ext == language,
                }
            })
        } else {
            false
        }
    }

    pub fn filter_supported_files(&self, paths: &[PathBuf]) -> Result<Vec<FileMetadata>> {
        let mut valid_files = Vec::new();

        for path in paths {
            match self.validate_file(path) {
                Ok(metadata) => {
                    if self.should_index_file(&metadata) {
                        valid_files.push(metadata);
                    }
                }
                Err(_) => {
                    // Skip files that can't be validated (permissions, etc.)
                    continue;
                }
            }
        }

        Ok(valid_files)
    }
}

impl Default for FileValidator {
    fn default() -> Self {
        Self::new()
    }
}