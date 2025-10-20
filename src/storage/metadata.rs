// ABOUTME: SQLite metadata storage for codesearch

use crate::error::Result;
use crate::models::embeddings::FileEmbedding;
use crate::parsers::Symbol;
use rusqlite::{params, Connection, OptionalExtension};
use std::path::{Path, PathBuf};
use std::time::SystemTime;

pub struct MetadataStorage {
    connection: Connection,
}

#[derive(Debug, Clone)]
pub struct ProjectMetadata {
    pub id: i64,
    pub path: PathBuf,
    pub hash: String,
    pub created_at: SystemTime,
    pub updated_at: SystemTime,
}

#[derive(Debug, Clone)]
pub struct FileMetadata {
    pub id: i64,
    pub project_id: i64,
    pub path: PathBuf,
    pub hash: String,
    pub size: u64,
    pub language: Option<String>,
    pub indexed_at: SystemTime,
    pub last_modified: SystemTime,
}

#[derive(Debug, Clone)]
pub struct ChunkMetadata {
    pub id: i64,
    pub file_id: i64,
    pub start_line: usize,
    pub end_line: usize,
    pub chunk_type: String,
    pub symbol_name: Option<String>,
    pub vector_offset: i64,
    pub created_at: SystemTime,
}

impl MetadataStorage {
    pub fn new<P: AsRef<Path>>(db_path: P) -> Result<Self> {
        let connection = Connection::open(db_path)?;
        let storage = Self { connection };
        storage.initialize_schema()?;
        Ok(storage)
    }

    pub fn initialize_schema(&self) -> Result<()> {
        self.connection.execute_batch(
            r#"
            CREATE TABLE IF NOT EXISTS projects (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                path TEXT UNIQUE NOT NULL,
                hash TEXT NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
            );

            CREATE TABLE IF NOT EXISTS files (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                project_id INTEGER NOT NULL,
                path TEXT NOT NULL,
                hash TEXT NOT NULL,
                size INTEGER NOT NULL,
                language TEXT,
                indexed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                last_modified DATETIME,
                FOREIGN KEY (project_id) REFERENCES projects (id) ON DELETE CASCADE,
                UNIQUE(project_id, path)
            );

            CREATE TABLE IF NOT EXISTS symbols (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                file_id INTEGER NOT NULL,
                name TEXT NOT NULL,
                kind TEXT NOT NULL,
                start_line INTEGER NOT NULL,
                end_line INTEGER NOT NULL,
                start_byte INTEGER NOT NULL,
                end_byte INTEGER NOT NULL,
                parent_symbol_id INTEGER,
                FOREIGN KEY (file_id) REFERENCES files (id) ON DELETE CASCADE,
                FOREIGN KEY (parent_symbol_id) REFERENCES symbols (id)
            );

            CREATE TABLE IF NOT EXISTS chunks (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                file_id INTEGER NOT NULL,
                start_line INTEGER NOT NULL,
                end_line INTEGER NOT NULL,
                chunk_type TEXT NOT NULL,
                symbol_name TEXT,
                vector_offset INTEGER NOT NULL,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (file_id) REFERENCES files (id) ON DELETE CASCADE
            );

            CREATE TABLE IF NOT EXISTS index_metadata (
                key TEXT PRIMARY KEY,
                value TEXT NOT NULL
            );

            CREATE INDEX IF NOT EXISTS idx_files_project_path ON files (project_id, path);
            CREATE INDEX IF NOT EXISTS idx_symbols_file_name ON symbols (file_id, name);
            CREATE INDEX IF NOT EXISTS idx_symbols_kind ON symbols (kind);
            CREATE INDEX IF NOT EXISTS idx_chunks_file_lines ON chunks (file_id, start_line);
            CREATE INDEX IF NOT EXISTS idx_chunks_symbol ON chunks (symbol_name);
            "#
        )?;
        Ok(())
    }

    pub fn create_or_update_project(&self, path: &Path, hash: &str) -> Result<i64> {
        let path_str = path.to_string_lossy();

        // Try to update existing project first
        let rows_updated = self.connection.execute(
            "UPDATE projects SET hash = ?1, updated_at = CURRENT_TIMESTAMP WHERE path = ?2",
            params![hash, path_str],
        )?;

        if rows_updated > 0 {
            // Project existed, get its ID
            let id: i64 = self.connection.query_row(
                "SELECT id FROM projects WHERE path = ?1",
                params![path_str],
                |row| row.get(0),
            )?;
            Ok(id)
        } else {
            // Create new project
            self.connection.execute(
                "INSERT INTO projects (path, hash) VALUES (?1, ?2)",
                params![path_str, hash],
            )?;
            Ok(self.connection.last_insert_rowid())
        }
    }

    pub fn get_project_by_path(&self, path: &Path) -> Result<Option<ProjectMetadata>> {
        let path_str = path.to_string_lossy();

        let project = self.connection.query_row(
            "SELECT id, path, hash, created_at, updated_at FROM projects WHERE path = ?1",
            params![path_str],
            |row| {
                Ok(ProjectMetadata {
                    id: row.get(0)?,
                    path: PathBuf::from(row.get::<_, String>(1)?),
                    hash: row.get(2)?,
                    created_at: chrono::DateTime::parse_from_rfc3339(&row.get::<_, String>(3)?)
                        .unwrap()
                        .with_timezone(&chrono::Utc)
                        .into(),
                    updated_at: chrono::DateTime::parse_from_rfc3339(&row.get::<_, String>(4)?)
                        .unwrap()
                        .with_timezone(&chrono::Utc)
                        .into(),
                })
            },
        ).optional()?;

        Ok(project)
    }

    pub fn create_or_update_file(&self, project_id: i64, file_path: &Path, hash: &str,
                               size: u64, language: Option<String>, last_modified: SystemTime) -> Result<i64> {
        let path_str = file_path.to_string_lossy();
        let lang_str = language.as_deref();
        let last_modified_str = chrono::DateTime::<chrono::Utc>::from(last_modified)
            .format("%Y-%m-%d %H:%M:%S%.3f").to_string();

        // Try to update existing file first
        let rows_updated = self.connection.execute(
            "UPDATE files SET hash = ?1, size = ?2, language = ?3, indexed_at = CURRENT_TIMESTAMP, last_modified = ?4
             WHERE project_id = ?5 AND path = ?6",
            params![hash, size, lang_str, last_modified_str, project_id, path_str],
        )?;

        if rows_updated > 0 {
            // File existed, get its ID
            let id: i64 = self.connection.query_row(
                "SELECT id FROM files WHERE project_id = ?1 AND path = ?2",
                params![project_id, path_str],
                |row| row.get(0),
            )?;
            Ok(id)
        } else {
            // Create new file
            self.connection.execute(
                "INSERT INTO files (project_id, path, hash, size, language, last_modified)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
                params![project_id, path_str, hash, size, lang_str, last_modified_str],
            )?;
            Ok(self.connection.last_insert_rowid())
        }
    }

    pub fn store_symbols(&self, file_id: i64, symbols: &[Symbol]) -> Result<()> {
        // Clear existing symbols for this file
        self.connection.execute(
            "DELETE FROM symbols WHERE file_id = ?1",
            params![file_id],
        )?;

        // Insert new symbols
        for symbol in symbols {
            let kind_str = format!("{:?}", symbol.kind);
            let parent_name = symbol.parent.as_deref();

            self.connection.execute(
                "INSERT INTO symbols (file_id, name, kind, start_line, end_line, start_byte, end_byte, parent_symbol_id)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8)",
                params![
                    file_id,
                    symbol.name,
                    kind_str,
                    symbol.start_line,
                    symbol.end_line,
                    symbol.start_byte,
                    symbol.end_byte,
                    parent_name,
                ],
            )?;
        }
        Ok(())
    }

    pub fn store_chunks(&self, file_id: i64, embeddings: &[FileEmbedding], start_offset: i64) -> Result<Vec<i64>> {
        let mut chunk_ids = Vec::new();

        // Clear existing chunks for this file
        self.connection.execute(
            "DELETE FROM chunks WHERE file_id = ?1",
            params![file_id],
        )?;

        // Insert new chunks
        for (i, embedding) in embeddings.iter().enumerate() {
            let chunk_type_str = format!("{:?}", embedding.chunk.chunk_type);
            let symbol_name = embedding.chunk.symbol_name.as_deref();
            let vector_offset = start_offset + i as i64;

            self.connection.execute(
                "INSERT INTO chunks (file_id, start_line, end_line, chunk_type, symbol_name, vector_offset)
                 VALUES (?1, ?2, ?3, ?4, ?5, ?6)",
                params![
                    file_id,
                    embedding.chunk.start_line,
                    embedding.chunk.end_line,
                    chunk_type_str,
                    symbol_name,
                    vector_offset,
                ],
            )?;

            chunk_ids.push(self.connection.last_insert_rowid());
        }

        Ok(chunk_ids)
    }

    pub fn find_symbols_by_name(&self, project_id: i64, symbol_name: &str) -> Result<Vec<(String, PathBuf, usize, usize)>> {
        let mut stmt = self.connection.prepare(
            r#"
            SELECT s.name, f.path, s.start_line, s.end_line
            FROM symbols s
            JOIN files f ON s.file_id = f.id
            WHERE f.project_id = ?1 AND s.name LIKE ?2
            ORDER BY s.name
            "#
        )?;

        let symbol_iter = stmt.query_map(
            params![project_id, format!("%{}%", symbol_name)],
            |row| {
                Ok((
                    row.get::<_, String>(0)?,
                    PathBuf::from(row.get::<_, String>(1)?),
                    row.get::<_, usize>(2)?,
                    row.get::<_, usize>(3)?,
                ))
            },
        )?;

        let mut results = Vec::new();
        for symbol in symbol_iter {
            results.push(symbol?);
        }
        Ok(results)
    }

    pub fn get_files_for_project(&self, project_id: i64) -> Result<Vec<FileMetadata>> {
        let mut stmt = self.connection.prepare(
            "SELECT id, project_id, path, hash, size, language, indexed_at, last_modified
             FROM files WHERE project_id = ?1"
        )?;

        let file_iter = stmt.query_map(
            params![project_id],
            |row| {
                Ok(FileMetadata {
                    id: row.get(0)?,
                    project_id: row.get(1)?,
                    path: PathBuf::from(row.get::<_, String>(2)?),
                    hash: row.get(3)?,
                    size: row.get(4)?,
                    language: row.get(5)?,
                    indexed_at: chrono::DateTime::parse_from_rfc3339(&row.get::<_, String>(6)?)
                        .unwrap()
                        .with_timezone(&chrono::Utc)
                        .into(),
                    last_modified: chrono::DateTime::parse_from_rfc3339(&row.get::<_, String>(7)?)
                        .unwrap()
                        .with_timezone(&chrono::Utc)
                        .into(),
                })
            },
        )?;

        let mut results = Vec::new();
        for file in file_iter {
            results.push(file?);
        }
        Ok(results)
    }

    pub fn get_chunks_for_search(&self, project_id: i64) -> Result<Vec<(i64, PathBuf, usize, usize, String, i64)>> {
        let mut stmt = self.connection.prepare(
            r#"
            SELECT c.id, f.path, c.start_line, c.end_line, c.chunk_type, c.vector_offset
            FROM chunks c
            JOIN files f ON c.file_id = f.id
            WHERE f.project_id = ?1
            ORDER BY c.id
            "#
        )?;

        let chunk_iter = stmt.query_map(
            params![project_id],
            |row| {
                Ok((
                    row.get::<_, i64>(0)?,
                    PathBuf::from(row.get::<_, String>(1)?),
                    row.get::<_, usize>(2)?,
                    row.get::<_, usize>(3)?,
                    row.get::<_, String>(4)?,
                    row.get::<_, i64>(5)?,
                ))
            },
        )?;

        let mut results = Vec::new();
        for chunk in chunk_iter {
            results.push(chunk?);
        }
        Ok(results)
    }

    pub fn delete_file(&self, project_id: i64, file_path: &Path) -> Result<bool> {
        let path_str = file_path.to_string_lossy();
        let rows_affected = self.connection.execute(
            "DELETE FROM files WHERE project_id = ?1 AND path = ?2",
            params![project_id, path_str],
        )?;
        Ok(rows_affected > 0)
    }

    pub fn get_statistics(&self, project_id: i64) -> Result<IndexStatistics> {
        let stats = self.connection.query_row(
            r#"
            SELECT
                COUNT(DISTINCT f.id) as total_files,
                COUNT(DISTINCT s.id) as total_symbols,
                COUNT(DISTINCT c.id) as total_chunks,
                SUM(f.size) as total_size,
                MAX(f.indexed_at) as last_indexed
            FROM files f
            LEFT JOIN symbols s ON f.id = s.file_id
            LEFT JOIN chunks c ON f.id = c.file_id
            WHERE f.project_id = ?1
            "#,
            params![project_id],
            |row| {
                Ok(IndexStatistics {
                    total_files: row.get::<_, i64>(0)? as usize,
                    total_symbols: row.get::<_, i64>(1)? as usize,
                    total_chunks: row.get::<_, i64>(2)? as usize,
                    total_size: row.get::<_, i64>(3)? as u64,
                    last_indexed: chrono::DateTime::parse_from_rfc3339(&row.get::<_, String>(4)?)
                        .unwrap()
                        .with_timezone(&chrono::Utc)
                        .into(),
                })
            },
        )?;
        Ok(stats)
    }
}

#[derive(Debug, Clone)]
pub struct IndexStatistics {
    pub total_files: usize,
    pub total_symbols: usize,
    pub total_chunks: usize,
    pub total_size: u64,
    pub last_indexed: SystemTime,
}