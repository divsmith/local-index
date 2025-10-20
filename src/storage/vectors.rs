// ABOUTME: Custom binary vector storage for codesearch

use crate::error::{CodeSearchError, Result};
use std::fs::{File, OpenOptions};
use std::io::{Read, Seek, SeekFrom, Write};
use std::path::Path;

#[derive(Debug, Clone)]
pub struct VectorFileHeader {
    pub magic: [u8; 4],
    pub version: u32,
    pub vector_count: u32,
    pub vector_dimension: u32,
    pub checksum: u64,
}

impl Default for VectorFileHeader {
    fn default() -> Self {
        Self {
            magic: *b"CSV\0", // CodeSearch Vector
            version: 1,
            vector_count: 0,
            vector_dimension: 768,
            checksum: 0,
        }
    }
}

pub struct VectorStorage {
    file: File,
    header: VectorFileHeader,
    file_path: std::path::PathBuf,
}

impl VectorStorage {
    /// Open existing vector storage or create new one
    pub fn open_or_create<P: AsRef<Path>>(path: P, vector_dimension: u32) -> Result<Self> {
        if path.as_ref().exists() {
            Self::open(path)
        } else {
            Self::create(path, vector_dimension)
        }
    }

    pub fn create<P: AsRef<Path>>(path: P, vector_dimension: u32) -> Result<Self> {
        let file_path = path.as_ref().to_path_buf();
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .truncate(true)
            .open(&file_path)?;

        let header = VectorFileHeader {
            magic: *b"CSV\0",
            version: 1,
            vector_count: 0,
            vector_dimension,
            checksum: 0,
        };

        let mut storage = Self {
            file,
            header,
            file_path,
        };

        storage.write_header()?;
        Ok(storage)
    }

    pub fn open<P: AsRef<Path>>(path: P) -> Result<Self> {
        let file_path = path.as_ref().to_path_buf();
        let mut file = OpenOptions::new()
            .read(true)
            .write(true)
            .open(&file_path)?;

        let header = Self::read_header(&mut file)?;

        // Validate magic bytes
        if header.magic != *b"CSV\0" {
            return Err(CodeSearchError::Storage(
                format!("Invalid vector file format: {:?}", file_path)
            ));
        }

        // Validate version
        if header.version != 1 {
            return Err(CodeSearchError::Storage(
                format!("Unsupported vector file version: {}", header.version)
            ));
        }

        Ok(Self {
            file,
            header,
            file_path,
        })
    }

    fn read_header(file: &mut File) -> Result<VectorFileHeader> {
        let mut magic = [0u8; 4];
        file.read_exact(&mut magic)?;

        let mut version_bytes = [0u8; 4];
        file.read_exact(&mut version_bytes)?;
        let version = u32::from_le_bytes(version_bytes);

        let mut count_bytes = [0u8; 4];
        file.read_exact(&mut count_bytes)?;
        let vector_count = u32::from_le_bytes(count_bytes);

        let mut dim_bytes = [0u8; 4];
        file.read_exact(&mut dim_bytes)?;
        let vector_dimension = u32::from_le_bytes(dim_bytes);

        let mut checksum_bytes = [0u8; 8];
        file.read_exact(&mut checksum_bytes)?;
        let checksum = u64::from_le_bytes(checksum_bytes);

        Ok(VectorFileHeader {
            magic,
            version,
            vector_count,
            vector_dimension,
            checksum,
        })
    }

    fn write_header(&mut self) -> Result<()> {
        self.file.seek(SeekFrom::Start(0))?;

        // Write magic bytes
        self.file.write_all(&self.header.magic)?;

        // Write version
        self.file.write_all(&self.header.version.to_le_bytes())?;

        // Write vector count
        self.file.write_all(&self.header.vector_count.to_le_bytes())?;

        // Write vector dimension
        self.file.write_all(&self.header.vector_dimension.to_le_bytes())?;

        // Write checksum (placeholder for now)
        self.file.write_all(&self.header.checksum.to_le_bytes())?;

        self.file.flush()?;
        Ok(())
    }

    pub fn append_vector(&mut self, vector: &[f32]) -> Result<u32> {
        if vector.len() != self.header.vector_dimension as usize {
            return Err(CodeSearchError::Storage(
                format!(
                    "Vector dimension mismatch: expected {}, got {}",
                    self.header.vector_dimension,
                    vector.len()
                )
            ));
        }

        // Seek to end of file
        self.file.seek(SeekFrom::End(0))?;

        // Write vector data
        for &value in vector {
            self.file.write_all(&value.to_le_bytes())?;
        }

        // Update header
        self.header.vector_count += 1;
        self.write_header()?;

        Ok(self.header.vector_count - 1) // Return offset of new vector
    }

    pub fn append_vectors(&mut self, vectors: &[Vec<f32>]) -> Result<Vec<u32>> {
        let mut offsets = Vec::with_capacity(vectors.len());

        for vector in vectors {
            let offset = self.append_vector(vector)?;
            offsets.push(offset);
        }

        Ok(offsets)
    }

    pub fn get_vector(&mut self, offset: u32) -> Result<Vec<f32>> {
        if offset >= self.header.vector_count {
            return Err(CodeSearchError::Storage(
                format!("Invalid vector offset: {}", offset)
            ));
        }

        let vector_size = std::mem::size_of::<f32>() * self.header.vector_dimension as usize;
        let file_offset = std::mem::size_of::<VectorFileHeader>() + offset as usize * vector_size;

        self.file.seek(SeekFrom::Start(file_offset as u64))?;

        let mut buffer = vec![0u8; vector_size];
        self.file.read_exact(&mut buffer)?;

        let vector: Vec<f32> = buffer.chunks_exact(4)
            .map(|chunk| f32::from_le_bytes([chunk[0], chunk[1], chunk[2], chunk[3]]))
            .collect();

        Ok(vector)
    }

    pub fn get_vectors(&mut self, offsets: &[u32]) -> Result<Vec<Vec<f32>>> {
        let mut vectors = Vec::with_capacity(offsets.len());

        for &offset in offsets {
            let vector = self.get_vector(offset)?;
            vectors.push(vector);
        }

        Ok(vectors)
    }

    pub fn update_vector(&mut self, offset: u32, vector: &[f32]) -> Result<()> {
        if vector.len() != self.header.vector_dimension as usize {
            return Err(CodeSearchError::Storage(
                format!(
                    "Vector dimension mismatch: expected {}, got {}",
                    self.header.vector_dimension,
                    vector.len()
                )
            ));
        }

        if offset >= self.header.vector_count {
            return Err(CodeSearchError::Storage(
                format!("Invalid vector offset: {}", offset)
            ));
        }

        let vector_size = std::mem::size_of::<f32>() * self.header.vector_dimension as usize;
        let file_offset = std::mem::size_of::<VectorFileHeader>() + offset as usize * vector_size;

        self.file.seek(SeekFrom::Start(file_offset as u64))?;

        // Write vector data
        for &value in vector {
            self.file.write_all(&value.to_le_bytes())?;
        }

        self.file.flush()?;
        Ok(())
    }

    pub fn vector_count(&self) -> u32 {
        self.header.vector_count
    }

    pub fn vector_dimension(&self) -> u32 {
        self.header.vector_dimension
    }

    pub fn file_size(&self) -> Result<u64> {
        let metadata = self.file.metadata()?;
        Ok(metadata.len())
    }

    pub fn sync(&mut self) -> Result<()> {
        self.file.sync_all()?;
        Ok(())
    }

    /// Get multiple vectors efficiently for similarity search
    pub fn get_vectors_for_search(&mut self, limit: Option<usize>) -> Result<Vec<(u32, Vec<f32>)>> {
        let count = limit.unwrap_or(self.header.vector_count as usize).min(self.header.vector_count as usize);
        let mut vectors = Vec::with_capacity(count);

        for offset in 0..count as u32 {
            let vector = self.get_vector(offset)?;
            vectors.push((offset, vector));
        }

        Ok(vectors)
    }

    /// Calculate file checksum (simplified version)
    pub fn calculate_checksum(&self) -> Result<u64> {
        // For now, return a simple hash based on file size and vector count
        let file_size = self.file_size()?;
        Ok(file_size.wrapping_mul(self.header.vector_count as u64))
    }

    /// Verify file integrity
    pub fn verify_integrity(&mut self) -> Result<bool> {
        // Check that the file size matches expected size
        let expected_size = std::mem::size_of::<VectorFileHeader>() as u64
            + (self.header.vector_count as u64 * self.header.vector_dimension as u64 * std::mem::size_of::<f32>() as u64);

        let actual_size = self.file_size()?;

        if actual_size != expected_size {
            return Err(CodeSearchError::Storage(
                format!("File size mismatch: expected {}, got {}", expected_size, actual_size)
            ));
        }

        // Verify magic bytes
        if self.header.magic != *b"CSV\0" {
            return Ok(false);
        }

        Ok(true)
    }
}

impl Drop for VectorStorage {
    fn drop(&mut self) {
        let _ = self.sync();
    }
}

