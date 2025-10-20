// ABOUTME: Index management and operations for codesearch

use crate::error::Result;
use crate::filesystem::FileValidator;
use crate::models::{EmbeddingGenerator, ModelManager, ModelConfig};
use crate::parsers::ParserRegistry;
use crate::storage::metadata::{MetadataStorage, IndexStatistics};
use crate::storage::vectors::VectorStorage;
use std::path::{Path, PathBuf};
use std::time::SystemTime;

pub struct IndexManager {
    project_root: PathBuf,
    metadata_storage: MetadataStorage,
    vector_storage: VectorStorage,
    parser_registry: ParserRegistry,
    model_manager: ModelManager,
    embedding_generator: EmbeddingGenerator,
    file_validator: FileValidator,
}

impl IndexManager {
    pub fn new<P: AsRef<Path>>(project_root: P) -> Result<Self> {
        let project_root = project_root.as_ref().to_path_buf();
        let index_dir = project_root.join(".codesearch");

        // Create index directory if it doesn't exist
        std::fs::create_dir_all(&index_dir)?;

        // Initialize storage components
        let metadata_storage = MetadataStorage::new(index_dir.join("metadata.db"))?;
        let vector_storage = VectorStorage::open_or_create(
            index_dir.join("vectors.dat"),
            ModelConfig::default().embedding_dimension as u32,
        )?;

        // Initialize processing components
        let parser_registry = ParserRegistry::new();
        let model_manager = ModelManager::new(ModelConfig::default())?;
        let embedding_generator = EmbeddingGenerator::new(model_manager.clone());
        let file_validator = FileValidator::new();

        Ok(Self {
            project_root,
            metadata_storage,
            vector_storage,
            parser_registry,
            model_manager,
            embedding_generator,
            file_validator,
        })
    }

    /// Open existing vector storage or create new one
    pub fn open_or_create<P: AsRef<Path>>(path: P, dimension: u32) -> Result<VectorStorage> {
        if path.as_ref().exists() {
            VectorStorage::open(path)
        } else {
            VectorStorage::create(path, dimension)
        }
    }

    pub fn index_directory(&self) -> Result<IndexProgress> {
        // Get or create project metadata
        let project_hash = self.calculate_project_hash()?;
        let project_id = self.metadata_storage.create_or_update_project(&self.project_root, &project_hash)?;

        // Scan directory for files
        let files = self.scan_directory()?;

        let progress = IndexProgress {
            total_files: files.len(),
            processed_files: 0,
            errors: Vec::new(),
        };

        self.index_files(project_id, files, progress)
    }

    fn index_files(&self, project_id: i64, files: Vec<PathBuf>, mut progress: IndexProgress) -> Result<IndexProgress> {
        for file_path in files {
            match self.index_single_file(project_id, &file_path) {
                Ok(_) => {
                    progress.processed_files += 1;
                }
                Err(e) => {
                    progress.errors.push(IndexError {
                        file_path: file_path.clone(),
                        error: e.to_string(),
                    });
                }
            }
        }

        Ok(progress)
    }

    fn index_single_file(&self, project_id: i64, file_path: &Path) -> Result<()> {
        // Validate file
        let file_metadata = self.file_validator.validate_file(file_path)?;
        if !self.file_validator.should_index_file(&file_metadata) {
            return Ok(()); // Skip files that shouldn't be indexed
        }

        // Read file content
        let content = std::fs::read_to_string(file_path)?;

        // Parse file and extract symbols
        let parse_result = self.parser_registry.parse_file(file_path, &content)?;

        // Generate embeddings
        let embeddings = self.embedding_generator.generate_file_embeddings(&content, &parse_result.symbols)?;

        // Get next vector offset
        let vector_offset = self.vector_storage.vector_count() as i64;

        // Store vectors
        let vector_data: Vec<Vec<f32>> = embeddings.iter().map(|e| e.embedding.clone()).collect();
        let mut storage = VectorStorage::open(
            self.project_root.join(".codesearch/vectors.dat")
        )?;
        storage.append_vectors(&vector_data)?;

        // Store metadata
        let language = file_metadata.language;
        let file_id = self.metadata_storage.create_or_update_file(
            project_id,
            file_path,
            &self.calculate_file_hash(&content)?,
            file_metadata.size,
            language,
            file_metadata.modified,
        )?;

        self.metadata_storage.store_symbols(file_id, &parse_result.symbols)?;
        self.metadata_storage.store_chunks(file_id, &embeddings, vector_offset)?;

        Ok(())
    }

    fn scan_directory(&self) -> Result<Vec<PathBuf>> {
        let mut files = Vec::new();

        for entry in walkdir::WalkDir::new(&self.project_root)
            .follow_links(false)
            .into_iter()
            .filter_entry(|e| !self.is_excluded(e.path()))
        {
            let entry = entry.map_err(|e| crate::error::CodeSearchError::Io(e.into()))?;

            if entry.file_type().is_file() {
                let path = entry.path().to_path_buf();
                if self.file_validator.validate_file(&path).is_ok() {
                    if let Some(_lang) = &self.file_validator.validate_file(&path).unwrap().language {
                        if self.parser_registry.is_file_supported(&path) {
                            files.push(path);
                        }
                    }
                }
            }
        }

        Ok(files)
    }

    fn is_excluded(&self, path: &Path) -> bool {
        let path_str = path.to_string_lossy();
        let exclude_patterns = [
            ".git", "target", "__pycache__", "node_modules", ".vscode", ".idea",
            ".codesearch", "Cargo.lock", "*.tmp", "*.log", "*.bak",
        ];

        exclude_patterns.iter().any(|pattern| {
            if pattern.starts_with("*.") {
                path_str.ends_with(&pattern[1..])
            } else {
                path_str.contains(pattern)
            }
        })
    }

    fn calculate_project_hash(&self) -> Result<String> {
        // Simple hash based on project root and current time
        use std::hash::{Hash, Hasher};
        use std::collections::hash_map::DefaultHasher;

        let mut hasher = DefaultHasher::new();
        self.project_root.hash(&mut hasher);
        SystemTime::now().hash(&mut hasher);

        Ok(format!("{:x}", hasher.finish()))
    }

    fn calculate_file_hash(&self, content: &str) -> Result<String> {
        use std::hash::{Hash, Hasher};
        use std::collections::hash_map::DefaultHasher;

        let mut hasher = DefaultHasher::new();
        content.hash(&mut hasher);

        Ok(format!("{:x}", hasher.finish()))
    }

    pub fn get_statistics(&self) -> Result<IndexStatistics> {
        let project = self.metadata_storage.get_project_by_path(&self.project_root)?
            .ok_or_else(|| crate::error::CodeSearchError::Search(
                "Project not found in index".to_string()
            ))?;

        self.metadata_storage.get_statistics(project.id)
    }

    pub fn rebuild_index(&self) -> Result<IndexProgress> {
        // Remove existing index
        let index_dir = self.project_root.join(".codesearch");
        if index_dir.exists() {
            std::fs::remove_dir_all(&index_dir)?;
        }
        std::fs::create_dir_all(&index_dir)?;

        // Rebuild from scratch
        self.index_directory()
    }

    pub fn incremental_index(&self) -> Result<IndexProgress> {
        // Get current project state
        let project = self.metadata_storage.get_project_by_path(&self.project_root)?;

        if let Some(project_meta) = project {
            // Scan for new/modified files
            let current_files = self.scan_directory()?;
            let indexed_files = self.metadata_storage.get_files_for_project(project_meta.id)?;

            let mut files_to_index = Vec::new();

            for file_path in current_files {
                let should_index = match indexed_files.iter().find(|f| f.path == file_path) {
                    Some(indexed_file) => {
                        // Check if file was modified
                        let current_metadata = std::fs::metadata(&file_path)?;
                        let current_modified = current_metadata.modified()?;
                        current_modified > indexed_file.last_modified
                    }
                    None => true, // New file
                };

                if should_index {
                    files_to_index.push(file_path);
                }
            }

            // Check for deleted files
            for indexed_file in &indexed_files {
                if !indexed_file.path.exists() {
                    self.metadata_storage.delete_file(project_meta.id, &indexed_file.path)?;
                }
            }

            // Index new/modified files
            self.index_files(project_meta.id, files_to_index, IndexProgress::new())
        } else {
            // No existing index, do full indexing
            self.index_directory()
        }
    }
}

#[derive(Debug, Clone)]
pub struct IndexProgress {
    pub total_files: usize,
    pub processed_files: usize,
    pub errors: Vec<IndexError>,
}

impl IndexProgress {
    pub fn new() -> Self {
        Self {
            total_files: 0,
            processed_files: 0,
            errors: Vec::new(),
        }
    }

    pub fn completion_rate(&self) -> f32 {
        if self.total_files == 0 {
            1.0
        } else {
            self.processed_files as f32 / self.total_files as f32
        }
    }

    pub fn is_complete(&self) -> bool {
        self.processed_files >= self.total_files
    }

    pub fn has_errors(&self) -> bool {
        !self.errors.is_empty()
    }
}

#[derive(Debug, Clone)]
pub struct IndexError {
    pub file_path: PathBuf,
    pub error: String,
}