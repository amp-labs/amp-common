# Immediate Improvement Opportunities

This document contains actionable improvements that can be made in just a few commits.

## Documentation Additions

- Missing CONTRIBUTING.md - Generate contributor guidelines based on existing code patterns
- Missing FAQ document - Create from common issues and patterns observed in the codebase

## Code Examples (High Priority Packages)

- future/ package - Add example_test.go based on existing README examples for godoc integration
- actor/ package - Add example_test.go showing basic actor patterns
- pool/ package - Add example_test.go demonstrating lifecycle management
- simultaneously/ package - Add example_test.go for common parallel execution patterns
- envutil/ package - Add example_test.go showing fluent API chaining and error handling

## Testing Documentation

- Add comment in CLAUDE.md explaining that t.Parallel() is required by linter for all tests

## Error Handling

- Audit error wrapping consistency (ensure %w is used instead of %v where appropriate)
- Add error handling note to CLAUDE.md: error classification should happen in the errors package, not in retry package

## Development Workflow

- Add .vscode/tours/ directory with code tours for key packages (similar to ~/src/server/.tours)
- Add GitHub issue templates (bug, feature, question)
- Add pull request template (description, testing checklist, documentation updates)
- Add .editorconfig for consistent formatting across editors

## Security & Best Practices

- Add SECURITY.md with security reporting guidelines
- Add documentation to envutil/ package about handling secrets safely

## Tooling Integration

- Add Dependabot or Renovate config for automated dependency updates

## Code Quality

- Review and address the 3 files with TODO/FIXME comments
