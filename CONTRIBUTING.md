# 🤝 Contributing Guidelines

Thanks for your interest in contributing! To keep things consistent and maintainable, please follow these simple rules when contributing to this project.

## Commit Messages

Use **Pascal Case** for commits in the following format:

```
Category: Summary of commit
```

### Examples:
- `Chore: Lint Codebase`
- `Commanding: Implement Cool Feature`
- `Fix: Resolve Timeout Error`
- `Docs: Update README`

**Categories** can include (but are not limited to):
- `Chore` – Routine tasks or refactoring
- `Fix` – Bug fixes
- `Feature` – New functionality
- `Docs` – Documentation changes
- `Commanding`, `Panel`, etc. – Custom areas of the code

---

## Branch Naming

Branch names should follow this format:

```
action/summary-of-branch
```


Use lowercase and hyphens for readability.

### Examples:
- `dev/add-login-panel`
- `fix/commanding-timeout`
- `refactor/multiplexer-cleanup`

## Pull Requests

- All PRs **must pass CI checks** before being reviewed.
- Keep PRs focused and atomic (one logical change per PR).
- Include relevant details in the PR description (what it does, why it matters).
- If you're fixing a bug or implementing a feature, reference the related issue if applicable.

## Code Style & Testing

- Match the existing code style.
- Write tests if you're adding new functionality or fixing bugs.
- Run tests locally before pushing if possible.

## Issues

- Use **Pascal Case** for issue titles as well:
  - Example: `Feature: Add Support For X`
  - Example: `Bug: Crash When Doing Y`

- Include steps to reproduce, expected vs. actual behavior, and any relevant logs or screenshots.

Thanks again! Let's make this project awesome together!
