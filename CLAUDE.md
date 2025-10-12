# clindex Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-06

## Active Technologies
- Go 1.24.6 + Standard library + existing index/search libraries (002-users-should-be)

## Project Structure
```
src/
tests/
```

## Commands
# Add commands for Go 1.24.6

## Code Style
Go 1.24.6: Follow standard conventions

## Recent Changes
- 002-users-should-be: Added Go 1.24.6 + Standard library + existing index/search libraries

<!-- MANUAL ADDITIONS START -->
## Session End Validation
Always validate that the build and tests pass successfully at the end of each working session:
- Run `go build ./...` to ensure all packages compile without errors
- Run `go test ./...` to ensure all tests pass
- Fix any build or test failures before concluding the session
<!-- MANUAL ADDITIONS END -->