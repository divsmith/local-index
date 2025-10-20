// ABOUTME: Data storage for codesearch

pub mod metadata;
pub mod vectors;
pub mod index;

pub use metadata::{MetadataStorage, ProjectMetadata, FileMetadata, ChunkMetadata, IndexStatistics};
pub use vectors::{VectorStorage, VectorFileHeader};
pub use index::{IndexManager, IndexProgress, IndexError};