
# Implementation Plan: Optional Directory Selection for Indexing and Searching

**Branch**: `002-users-should-be` | **Date**: 2025-10-06 | **Spec**: `/specs/002-users-should-be/spec.md`
**Input**: Feature specification from `/specs/002-users-should-be/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
This feature adds optional directory selection to the code-search CLI tool, allowing users to specify target directories for both indexing and searching operations. The technical approach uses Go's cross-platform file handling, stores index files in a `.clindex` subdirectory within the target directory, maintains backward compatibility with current directory as default, and implements comprehensive validation and error handling for directory operations.

## Technical Context
**Language/Version**: Go 1.24.6
**Primary Dependencies**: Standard library + existing index/search libraries
**Storage**: File system (index files stored alongside source content)
**Testing**: Go testing framework with contract and integration tests
**Target Platform**: Cross-platform CLI tool (Linux, macOS, Windows)
**Project Type**: Single CLI application
**Performance Goals**: Fast indexing and search operations, sub-second response times for typical directories
**Constraints**: Must handle large directories efficiently, memory-efficient indexing, backward compatibility required
**Scale/Scope**: Support directories with thousands of files, configurable size limits

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Code Quality First**: ✅ PASS - Feature includes comprehensive testing requirements, will maintain code quality standards
**Modular Architecture**: ✅ PASS - Directory selection will be implemented as modular command-line options with clear interfaces
**Test-Driven Development**: ✅ PASS - TDD approach will be strictly enforced with contract tests written before implementation
**Documentation-Driven**: ✅ PASS - CLI help and usage documentation will be comprehensive
**Security-First**: ✅ PASS - Input validation for directory paths and permission checks will be implemented

**Additional Constraints Check**:
- Industry standards: ✅ PASS - Using Go standard library practices
- Dependency security: ✅ PASS - Minimal dependencies, standard library focused
- Performance benchmarks: ✅ PASS - Performance goals defined in technical context
- Error handling: ✅ PASS - CLI error handling patterns will be followed

**Development Workflow Check**:
- Peer review ready: ✅ PASS - Design will be reviewable
- Automated tests: ✅ PASS - Contract and integration tests planned
- Code coverage: ✅ PASS - Tests will drive implementation
- Commit format: ✅ PASS - Will follow conventional commits

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
src/
├── main.go              # Application entry point
├── app.go               # Application configuration and setup
├── errors.go            # Error handling definitions
├── index_cmd.go         # Index command implementation (to be modified)
├── search_cmd.go        # Search command implementation (to be modified)
├── models/              # Data models and structures
│   ├── index.go         # Index-related models
│   └── search.go        # Search-related models
├── services/            # Business logic services
│   ├── indexer.go       # Indexing service (to be modified)
│   └── searcher.go      # Search service (to be modified)
└── lib/                 # Shared libraries and utilities
    ├── fileutils.go     # File system utilities (to be enhanced)
    └── validation.go    # Input validation utilities (to be added)

tests/
├── contract/            # Contract tests for CLI commands
├── integration/         # Integration tests for end-to-end flows
└── unit/                # Unit tests for individual components
```

**Structure Decision**: Single Go CLI application with existing structure. The feature will be implemented by extending the current CLI commands (`index_cmd.go` and `search_cmd.go`) to support optional directory selection parameters, enhancing the services layer for directory handling, and adding validation utilities for path processing.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh claude`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract → contract test task [P]
- Each entity → model creation task [P]
- Each user story → integration test task
- Implementation tasks to make tests pass

**Specific Tasks to Generate**:
1. **Model Implementation** (`src/lib/validation.go`, update `src/models/`)
2. **CLI Enhancement** (`src/index_cmd.go`, `src/search_cmd.go`)
3. **Service Layer Updates** (`src/services/indexer.go`, `src/services/searcher.go`)
4. **File Utilities** (`src/lib/fileutils.go`)
5. **Contract Tests** (CLI command validation)
6. **Integration Tests** (end-to-end workflows)
7. **Error Handling** (comprehensive error scenarios)

**Ordering Strategy**:
- TDD order: Tests before implementation
- Dependency order: Models before services before CLI
- Mark [P] for parallel execution (independent files)

**Estimated Output**: 25-30 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
