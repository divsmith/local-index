# Tasks: High Performance Local CLI Vectorized Codebase Search

**Input**: Design documents from `/specs/001-high-performance-local/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: tech stack, libraries, structure
2. Load optional design documents:
   → data-model.md: Extract entities → model tasks
   → contracts/: Each file → contract test task
   → research.md: Extract decisions → setup tasks
3. Generate tasks by category:
   → Setup: project init, dependencies, linting
   → Tests: contract tests, integration tests
   → Core: models, services, CLI commands
   → Integration: DB, middleware, logging
   → Polish: unit tests, performance, docs
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → All contracts have tests?
   → All entities have models?
   → All endpoints implemented?
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Single project**: `src/`, `tests/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

## Phase 3.1: Setup
- [ ] T001 Create project structure per implementation plan with src/models/, src/services/, src/cli/, src/lib/, tests/
- [ ] T002 Initialize Go project with go.mod for the CLI tool
- [ ] T003 [P] Configure Go linting and formatting tools (golint, gofmt)

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

- [ ] T004 [P] Contract test for CLI index command in tests/contract/test_cli_index.go
- [ ] T005 [P] Contract test for CLI search command in tests/contract/test_cli_search.go
- [ ] T006 [P] Integration test for search functionality in tests/integration/test_search_integration.go
- [ ] T007 [P] Integration test for indexing functionality in tests/integration/test_indexing_integration.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### Models
- [ ] T008 [P] CodeIndex model in src/models/code_index.go
- [ ] T009 [P] FileEntry model in src/models/file_entry.go
- [ ] T010 [P] CodeChunk model in src/models/code_chunk.go
- [ ] T011 [P] SearchQuery model in src/models/search_query.go
- [ ] T012 [P] SearchResult model in src/models/search_result.go
- [ ] T013 [P] SearchResults model in src/models/search_results.go

### Services
- [ ] T014 [P] IndexingService in src/services/indexing_service.go
- [ ] T015 [P] SearchService in src/services/search_service.go

### CLI Commands
- [ ] T016 [P] CLI main entry point in src/cli/main.go
- [ ] T017 [P] Search command implementation in src/cli/search_cmd.go
- [ ] T018 [P] Index command implementation in src/cli/index_cmd.go

### Libraries
- [ ] T019 [P] Vector database wrapper in src/lib/vector_db.go
- [ ] T020 [P] File system scanner in src/lib/file_scanner.go
- [ ] T021 [P] Code parsing utilities in src/lib/parser.go

## Phase 3.4: Integration
- [ ] T022 Connect IndexingService to file scanner and vector DB
- [ ] T023 Connect SearchService to vector DB for search operations
- [ ] T024 Implement CLI error handling and logging
- [ ] T025 Add input validation for CLI commands and file paths

## Phase 3.5: Polish
- [ ] T026 [P] Unit tests for CodeIndex model in tests/unit/models/code_index_test.go
- [ ] T027 [P] Unit tests for FileEntry model in tests/unit/models/file_entry_test.go
- [ ] T028 [P] Unit tests for CodeChunk model in tests/unit/models/code_chunk_test.go
- [ ] T029 [P] Unit tests for SearchQuery model in tests/unit/models/search_query_test.go
- [ ] T030 [P] Unit tests for SearchResult model in tests/unit/models/search_result_test.go
- [ ] T031 [P] Unit tests for SearchResults model in tests/unit/models/search_results_test.go
- [ ] T032 [P] Unit tests for IndexingService in tests/unit/services/indexing_service_test.go
- [ ] T033 [P] Unit tests for SearchService in tests/unit/services/search_service_test.go
- [ ] T034 [P] Unit tests for CLI commands in tests/unit/cli/search_cmd_test.go
- [ ] T035 [P] Performance tests for search functionality in tests/performance/search_performance_test.go
- [ ] T036 [P] Update documentation in README.md with usage instructions
- [ ] T037 Remove code duplication and optimize for memory usage
- [ ] T038 Run manual verification steps from quickstart guide

## Dependencies
- Setup (T001-T003) before tests (T004-T007)
- Tests (T004-T033) before implementation (T008-T025)
- Models (T008-T013) before services (T014-T015)
- Services (T014-T015) before CLI commands (T016-T018)
- Core implementation (T008-T021) before integration (T022-T025)
- Implementation before polish (T026-T038)

## Parallel Example
```
# Launch T004-T007 together:
Task: "Contract test for CLI index command in tests/contract/test_cli_index.go"
Task: "Contract test for CLI search command in tests/contract/test_cli_search.go"
Task: "Integration test for search functionality in tests/integration/test_search_integration.go"
Task: "Integration test for indexing functionality in tests/integration/test_indexing_integration.go"

# Launch T008-T013 together (models):
Task: "CodeIndex model in src/models/code_index.go"
Task: "FileEntry model in src/models/file_entry.go"
Task: "CodeChunk model in src/models/code_chunk.go"
Task: "SearchQuery model in src/models/search_query.go"
Task: "SearchResult model in src/models/search_result.go"
Task: "SearchResults model in src/models/search_results.go"

# Launch T026-T031 together (model unit tests):
Task: "Unit tests for CodeIndex model in tests/unit/models/code_index_test.go"
Task: "Unit tests for FileEntry model in tests/unit/models/file_entry_test.go"
Task: "Unit tests for CodeChunk model in tests/unit/models/code_chunk_test.go"
Task: "Unit tests for SearchQuery model in tests/unit/models/search_query_test.go"
Task: "Unit tests for SearchResult model in tests/unit/models/search_result_test.go"
Task: "Unit tests for SearchResults model in tests/unit/models/search_results_test.go"
```

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing
- Commit after each task
- Avoid: vague tasks, same file conflicts

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - Each contract file → contract test task [P]
   - Each CLI command → implementation task
   
2. **From Data Model**:
   - Each entity → model creation task [P]
   - Relationships → service layer tasks
   
3. **From User Stories**:
   - Each story → integration test [P]
   - Quickstart scenarios → validation tasks

4. **Ordering**:
   - Setup → Tests → Models → Services → CLI → Integration → Polish
   - Dependencies block parallel execution

## Validation Checklist
*GATE: Checked by main() before returning*

- [ ] All contracts have corresponding tests
- [ ] All entities have model tasks
- [ ] All tests come before implementation
- [ ] Parallel tasks truly independent
- [ ] Each task specifies exact file path
- [ ] No task modifies same file as another [P] task