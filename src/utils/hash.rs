// ABOUTME: Hash and checksum utilities

use std::hash::{Hash, Hasher};
use std::collections::hash_map::DefaultHasher;

pub fn calculate_file_hash(path: &std::path::Path) -> Result<String, std::io::Error> {
    let content = std::fs::read(path)?;
    Ok(format!("{:x}", md5::compute(content)))
}

pub fn calculate_content_hash(content: &[u8]) -> String {
    format!("{:x}", md5::compute(content.to_vec()))
}

pub fn calculate_string_hash(s: &str) -> u64 {
    let mut hasher = DefaultHasher::new();
    s.hash(&mut hasher);
    hasher.finish()
}

// Simple MD5 implementation for testing
mod md5 {
    use std::collections::hash_map::DefaultHasher;
    use std::hash::{Hash, Hasher};

    pub struct Md5Digest([u8; 16]);

    impl Md5Digest {
        pub fn as_bytes(&self) -> &[u8; 16] {
            &self.0
        }
    }

    impl std::fmt::LowerHex for Md5Digest {
        fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            for byte in &self.0 {
                write!(f, "{:02x}", byte)?;
            }
            Ok(())
        }
    }

    pub fn compute(data: Vec<u8>) -> Md5Digest {
        // Placeholder MD5 implementation using simple hashing
        let mut hasher = DefaultHasher::new();
        data.hash(&mut hasher);
        let hash = hasher.finish();

        // Convert to 16-byte array (simplified)
        let mut bytes = [0u8; 16];
        for (i, chunk) in hash.to_le_bytes().iter().enumerate() {
            if i < 16 {
                bytes[i] = *chunk;
            }
        }

        Md5Digest(bytes)
    }
}