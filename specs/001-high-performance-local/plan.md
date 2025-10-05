# Implementation Plan: High Performance Local CLI Vectorized Codebase Search

**Branch**: `001-high-performance-local` | **Date**: 2025-10-04 | **Spec**: [/app/specs/001-high-performance-local/spec.md](file:///app/specs/001-high-performance-local/spec.md)
**Input**: Feature specification from `/specs/001-high-performance-local/spec.md`

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
The feature enables developers to perform high-performance vectorized searches on local codebases via a CLI interface. The approach will use Go with an embedded vector database to achieve fast search performance and relevance ranking. The system will recursively scan code files, index them using vector embeddings, and provide search results with context, file locations, and line numbers.

## Technical Context
**Language/Version**: Go 1.21
**Primary Dependencies**: An embedded vector database solution (e.g., Golang vector libraries for vectorized search)  
**Storage**: Local embedded vector database for indexing code, temporary search result caching
**Testing**: Go's built-in testing framework for unit and integration tests
**Target Platform**: Cross-platform CLI tool (Linux, macOS, Windows)
**Project Type**: Single CLI application with embedded indexing and search capabilities
**Performance Goals**: Response under 2 seconds for repositories up to 100,000 lines, under 10 seconds for repositories up to 1,000,000 lines
**Constraints**: Memory efficient (<500MB for typical usage), fast indexing and search, minimal CPU usage during indexing
**Scale/Scope**: Local codebase search up to 1 million lines of code

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Based on the Qwen Code Constitution:
- Code Quality First: Implementation will include comprehensive unit tests
- Modular Architecture: Components will have well-defined interfaces between indexing, search, and CLI modules
- Test-Driven Development: Tests will be written before implementation following Red-Green-Refactor cycle
- Documentation-Driven: All code will be documented before merging with clear comments explaining the 'why'
- Security-First: Input validation will be implemented for all user inputs and file paths

## Project Structure

### Documentation (this feature)
```
specs/001-high-performance-local/
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
├── models/
│   ├── code_index.go      # Code indexing structures and logic
│   ├── search_query.go    # Search query processing
│   └── search_result.go   # Search result structure
├── services/
│   ├── indexing_service.go  # Handles codebase indexing
│   └── search_service.go    # Performs vectorized search operations
├── cli/
│   ├── main.go              # CLI entry point
│   ├── search_cmd.go        # Search command implementation
│   └── index_cmd.go         # Index command implementation
└── lib/
    ├── vector_db.go         # Vector database wrapper
    ├── file_scanner.go      # File system scanner
    └── parser.go            # Code parsing utilities

tests/
├── contract/
├── integration/
│   ├── search_integration_test.go
│   └── indexing_integration_test.go
└── unit/
    ├── models/
    │   ├── code_index_test.go
    │   └── search_query_test.go
    ├── services/
    │   ├── indexing_service_test.go
    │   └── search_service_test.go
    └── cli/
        └── search_cmd_test.go
```

**Structure Decision**: Single CLI project structure with well-separated concerns between models, services, CLI interface, and utility libraries. This structure enables modular development, independent testing of components, and follows Go project conventions.

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
   - Run `.specify/scripts/bash/update-agent-context.sh qwen`
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

**Ordering Strategy**:
- TDD order: Tests before implementation 
- Dependency order: Models before services before UI
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
*Based on Constitution v1.0.0 - See `/memory/constitution.md`*