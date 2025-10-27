# ONNX Integration for CodeSearch

This document describes the ONNX integration implementation and setup requirements for the CodeSearch project.

## Overview

The CodeSearch project now includes real ONNX-based semantic embeddings, replacing the previous mock-only implementation. This provides actual semantic understanding of code rather than pattern-based heuristics.

## Architecture

### Components

1. **OnnxEmbeddingGenerator** (`src/models/onnx_embeddings.rs`)
   - Core ONNX inference engine
   - Model loading and session management
   - Batch and single text embedding generation
   - Automatic model downloading and caching

2. **ModelManager** (`src/models/mod.rs`)
   - Unified interface for both ONNX and fallback embeddings
   - Automatic fallback to mock implementation when ONNX unavailable
   - Async model loading and embedding generation

3. **FallbackModelManager** (`src/models/fallback.rs`)
   - Enhanced mock implementation for environments without ONNX
   - Semantic feature extraction based on code patterns
   - Maintains backward compatibility

### Supported Models

- **CodeBERT-base**: Microsoft's code understanding model
- **UniXcoder-base**: Multilingual code model
- **GraphCodeBERT-base**: Code with graph structure understanding

## Setup Requirements

### System Dependencies

The ONNX integration requires:

1. **C Compiler**: `gcc` or `clang` (required by Rust dependencies)
2. **ONNX Runtime**: Automatically included via Cargo
3. **Rust 1.70+**: For async/await and tokio support

### Platform-Specific Setup

#### Linux (Ubuntu/Debian)
```bash
sudo apt-get update
sudo apt-get install build-essential pkg-config
```

#### Linux (Alpine)
```bash
sudo apk add build-base pkgconfig
```

#### macOS
```bash
xcode-select --install
```

#### Windows
Install Visual Studio Build Tools or Visual Studio Community with C++ development tools.

### Environment Variables

- `HOME`: Used for model cache directory (`~/.codesearch/models`)
- Model cache can be overridden by modifying `ModelManager::new()`

## Usage

### Basic Usage

```rust
use codesearch::models::{ModelManager, ModelConfig};

#[tokio::main]
async fn main() -> codesearch::error::Result<()> {
    let config = ModelConfig::default();
    let mut manager = ModelManager::new(config)?;

    // Generate embeddings (automatically tries ONNX first)
    let texts = vec![
        "fn hello_world() { println!(\"Hello!\"); }".to_string(),
        "class MyClass { fn new() -> Self { Self {} } }".to_string(),
    ];

    let embeddings = manager.generate_embeddings(&texts).await?;
    println!("Generated {} embeddings", embeddings.len());

    Ok(())
}
```

### Fallback Behavior

The system automatically falls back to mock embeddings when:
- ONNX models are not available
- Model download fails
- ONNX runtime initialization fails

This ensures the application works in any environment.

## Testing

### Running Tests

```bash
# Run all tests (works in any environment)
cargo test

# Run ONNX-specific tests (requires ONNX setup)
cargo test onnx_specific_tests

# Run benchmarks
cargo test benchmarks
```

### Test Coverage

The test suite includes:
- **Fallback embedding tests**: Always available
- **ONNX integration tests**: Require ONNX setup
- **Performance benchmarks**: Validate speed requirements
- **Consistency tests**: Ensure deterministic behavior

## Performance

### Expected Performance

- **Fallback embeddings**: ~10ms per text
- **ONNX embeddings**: ~50-100ms per text (first load slower due to model download)
- **Batch processing**: ~2-5x faster than individual calls

### Memory Usage

- **Model cache**: ~200-500MB per model
- **Runtime memory**: ~100MB additional
- **Fallback implementation**: <10MB

## Troubleshooting

### Common Issues

1. **Linker errors**: Install C compiler (gcc/clang)
2. **Model download failures**: Check internet connection and HuggingFace availability
3. **Out of memory**: Reduce batch size or use smaller models
4. **Slow first load**: Models are downloaded and cached on first use

### Debug Mode

Set `RUST_LOG=debug` to see detailed ONNX operation logs:

```bash
RUST_LOG=debug cargo test
```

## Development

### Adding New Models

1. Update `ModelManager::load_model()` with new model URLs
2. Add model-specific configuration if needed
3. Update tests to include new model validation

### Custom Embedding Dimensions

Modify `ModelConfig::default()` or provide custom config:

```rust
let config = ModelConfig {
    embedding_dimension: 1024,  // Custom dimension
    ..Default::default()
};
```

## Future Enhancements

1. **GPU Support**: Enable CUDA/OpenVINO execution providers
2. **Model Quantization**: Support for INT8/FP16 models
3. **Dynamic Batching**: Automatic batch size optimization
4. **Local Model Serving**: Standalone model server mode

## Platform Limitations

### Current Environment Issues

The current aarch64-alpine-linux-musl environment has compilation limitations:
- Missing C compiler (`cc` not found)
- Incompatible toolchain for native dependencies
- Limited package availability

### Solutions

1. **Use standard Linux environment**: Ubuntu/Debian/CentOS
2. **Install build tools**: `build-essential` or equivalent
3. **Cross-compilation**: Build on x86_64 and deploy binary
4. **Containerized environment**: Docker with proper toolchain

### Verification

To verify your environment supports ONNX integration:

```bash
# Check C compiler availability
gcc --version

# Check Rust toolchain
rustc --version

# Test basic compilation
cargo check
```

If all commands succeed, your environment should support ONNX integration.