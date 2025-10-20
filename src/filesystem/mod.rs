// ABOUTME: File system operations for codesearch

pub mod scanner;
pub mod validator;
pub mod watcher;

pub use scanner::FileScanner;
pub use validator::{FileValidator, FileMetadata};
pub use watcher::{FileWatcher, FileChangeEvent, DebouncedFileWatcher};