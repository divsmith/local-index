<!-- 
Sync Impact Report:
Version change: N/A → 1.0.0
Added sections: All principles (Code Quality First, Modular Architecture, Test-Driven Development, Documentation-Driven, Security-First), Additional Constraints, Development Workflow, and Governance sections
Removed sections: None
Templates requiring updates: ✅ no changes needed - .specify/templates/plan-template.md, .specify/templates/spec-template.md, .specify/templates/tasks-template.md (templates reference constitution generally but don't implement specific principles)
Follow-up TODOs: None
-->
# Qwen Code Constitution

## Core Principles

### Code Quality First
All code must meet high quality standards; Every feature must include comprehensive unit tests; Code reviews are mandatory before merging to ensure maintainability, readability, and performance.

### Modular Architecture
Components must have well-defined interfaces; Dependencies between modules should be minimal and clearly documented; Each module must be independently testable and deployable where possible.

### Test-Driven Development (NON-NEGOTIABLE)
TDD mandatory: Tests written → Tests fail → Then implement; Red-Green-Refactor cycle strictly enforced; New features must not break existing tests.

### Documentation-Driven
All code must be documented before merging; APIs must have comprehensive documentation; Comments must explain the 'why' not just the 'what'; README files must be updated with any significant changes.

### Security-First
Security considerations must be addressed at the design phase; Input validation is mandatory for all user inputs; All components must follow security best practices; Vulnerability assessments required before production releases.

## Additional Constraints

Technology stack must follow industry standards; All dependencies must be regularly updated and audited for security vulnerabilities; Performance benchmarks must be maintained or improved with each release; All external APIs must have fallback mechanisms.

## Development Workflow

All code changes must go through peer review; Pull requests must pass all automated tests before merging; Code coverage must not decrease below 80%; All commits must follow conventional commit format.

## Governance

Constitutional compliance must be verified during code reviews; Amendments to this constitution require team consensus and documented approval; This constitution supersedes all other development practices in case of conflicts.

**Version**: 1.0.0 | **Ratified**: 2025-10-04 | **Last Amended**: 2025-10-04