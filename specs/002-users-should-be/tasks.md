# Tasks: Optional Directory Selection for Indexing and Searching

**Input**: Design documents from `/specs/002-users-should-be/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Phase 3.1: Setup
- [ ] T001 [P] Create validation utilities in `src/lib/validation.go`
- [ ] T002 [P] Configure Go module dependencies for file handling
- [ ] T003 [P] Set up error handling structures in `src/errors.go`

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
- [ ] T004 [P] Contract test for index command with --dir flag in `tests/contract/test_index_dir.go`
- [ ] T005 [P] Contract test for search command with --dir flag in `tests/contract/test_search_dir.go`
- [ ] T006 [P] Contract test for directory validation in `tests/contract/test_directory_validation.go`
- [ ] T007 [P] Integration test for project setup workflow in `tests/integration/test_project_workflow.go`
- [ ] T008 [P] Integration test for multi-project workflow in `tests/integration/test_multi_project.go`
- [ ] T009 [P] Integration test for error scenarios in `tests/integration/test_error_scenarios.go`

## Phase 3.3: Core Implementation (ONLY after tests are failing)
- [ ] T010 [P] DirectoryConfig model in `src/models/index.go`
- [ ] T011 [P] DirectoryPerms model in `src/models/index.go`
- [ ] T012 [P] DirectoryLimits model in `src/models/index.go`
- [ ] T013 [P] DirectoryMetadata model in `src/models/index.go`
- [ ] T014 [P] IndexLocation model in `src/models/index.go`
- [ ] T015 [P] Directory validation service in `src/lib/validation.go`
- [ ] T016 [P] Path resolution utilities in `src/lib/fileutils.go`
- [ ] T017 [P] Update indexer service for directory selection in `src/services/indexer.go`
- [ ] T018 [P] Update searcher service for directory selection in `src/services/searcher.go`
- [ ] T019 Add --dir flag to index command in `src/index_cmd.go`
- [ ] T020 Add --dir flag to search command in `src/search_cmd.go`
- [ ] T021 Error handling for directory operations in `src/errors.go`

## Phase 3.4: Integration
- [ ] T022 Connect index command to directory validation
- [ ] T023 Connect search command to index location resolution
- [ ] T024 Add index file locking mechanism
- [ ] T025 Integrate path resolution across CLI commands
- [ ] T026 Add directory metadata tracking

## Phase 3.5: Polish
- [ ] T027 [P] Unit tests for validation in `tests/unit/test_validation.go`
- [ ] T028 [P] Unit tests for file utilities in `tests/unit/test_fileutils.go`
- [ ] T029 [P] Unit tests for models in `tests/unit/test_models.go`
- [ ] T030 Performance tests for large directories (<1s indexing)
- [ ] T031 [P] Update CLI help documentation
- [ ] T032 [P] Update README.md with new examples
- [ ] T033 Add verbose output for indexing progress
- [ ] T034 Run quickstart.md validation scenarios

## Dependencies
- Tests (T004-T009) before implementation (T010-T026)
- Models (T010-T014) block services (T017-T018)
- Services (T017-T018) block CLI commands (T019-T020)
- Integration (T022-T026) before polish (T027-T034)

## Parallel Execution Examples

### Phase 3.2 - Test Creation (Can run together)
```
Task: "Contract test for index command with --dir flag in tests/contract/test_index_dir.go"
Task: "Contract test for search command with --dir flag in tests/contract/test_search_dir.go"
Task: "Contract test for directory validation in tests/contract/test_directory_validation.go"
Task: "Integration test for project setup workflow in tests/integration/test_project_workflow.go"
Task: "Integration test for multi-project workflow in tests/integration/test_multi_project.go"
Task: "Integration test for error scenarios in tests/integration/test_error_scenarios.go"
```

### Phase 3.3 - Model Implementation (Can run together)
```
Task: "DirectoryConfig model in src/models/index.go"
Task: "DirectoryPerms model in src/models/index.go"
Task: "DirectoryLimits model in src/models/index.go"
Task: "DirectoryMetadata model in src/models/index.go"
Task: "IndexLocation model in src/models/index.go"
```

### Phase 3.3 - Utility Implementation (Can run together)
```
Task: "Directory validation service in src/lib/validation.go"
Task: "Path resolution utilities in src/lib/fileutils.go"
```

### Phase 3.5 - Unit Tests (Can run together)
```
Task: "Unit tests for validation in tests/unit/test_validation.go"
Task: "Unit tests for file utilities in tests/unit/test_fileutils.go"
Task: "Unit tests for models in tests/unit/test_models.go"
Task: "Update CLI help documentation"
Task: "Update README.md with new examples"
```

## Task Details by Contract

### CLI Interfaces Contract → Tasks
- Index command with --dir flag → T004 (test), T019 (implementation)
- Search command with --dir flag → T005 (test), T020 (implementation)
- Directory validation → T006 (test), T015 (implementation)
- Error responses → T009 (test), T021 (implementation)

### Data Model Contract → Tasks
- DirectoryConfig entity → T010 (model)
- DirectoryPerms entity → T011 (model)
- DirectoryLimits entity → T012 (model)
- DirectoryMetadata entity → T013 (model)
- IndexLocation entity → T014 (model)

### Quickstart Scenarios → Tasks
- Project setup workflow → T007 (test), T022 (integration)
- Multi-project workflow → T008 (test), T023 (integration)
- Error scenarios → T009 (test), T024-T026 (integration)

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing
- Commit after each task
- Each task specifies exact file path
- No task modifies same file as another [P] task
- Feature name: "Optional Directory Selection for Indexing and Searching"
- Total tasks: 34