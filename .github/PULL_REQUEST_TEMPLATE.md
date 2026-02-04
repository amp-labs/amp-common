# Description

<!-- Provide a clear and concise description of your changes -->

## Type of Change

<!-- Mark the relevant option with an 'x' -->

- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update
- [ ] Refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test additions or updates

## Testing Checklist

<!-- Ensure all items are checked before requesting review -->

- [ ] Tests added or updated to cover changes
- [ ] All tests call `t.Parallel()` (required by linter)
- [ ] `make fix` passes (formatting and auto-fixable linter issues)
- [ ] `make lint` passes (all linter checks)
- [ ] `make test` passes (all tests succeed)
- [ ] `make race` passes (if applicable - race detection for concurrency changes)

## Documentation

<!-- Mark applicable items -->

- [ ] Code comments added/updated for non-obvious logic
- [ ] Package documentation (godoc) updated
- [ ] README.md updated (if applicable)
- [ ] Example tests added (for new packages or significant features)
- [ ] CLAUDE.md updated (if adding new package or changing architecture)

## Breaking Changes

<!-- If you marked "Breaking change" above, describe the impact and migration path -->

**Does this PR introduce breaking changes?** No / Yes

If yes, describe:
- What breaks
- Why the breaking change is necessary
- How users should update their code

## Related Issues

<!-- Link related issues using keywords: Fixes #123, Closes #456, Relates to #789 -->

Fixes #

## Additional Context

<!-- Add any other context, screenshots, or information that would help reviewers -->

## Checklist Before Requesting Review

- [ ] Code follows the style guidelines in [CONTRIBUTING.md](CONTRIBUTING.md)
- [ ] Self-reviewed my own code
- [ ] Branch is up-to-date with main
- [ ] No merge conflicts
- [ ] Commit messages are clear and follow conventions
- [ ] PR title is clear and descriptive
