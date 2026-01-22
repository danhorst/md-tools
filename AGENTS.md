## Purpose

This document defines mandatory constraints and design principles for autonomous agents contributing code to this repository.

The goal of this project is to provide small, composable command-line tools for Markdown manipulation that adhere strictly to the Unix philosophy:

- Do one thing
- Do it well
- Compose cleanly with other tools

This file is written for agents, not humans. A separate README.md will explain usage and motivation for human readers.

Target Domain

- Markdown flavor: GitHub Flavored Markdown (GFM)
- Primary use case: Programmatic editing of Markdown documents via CLI filters
- Non-goals (explicit):
  - General-purpose Markdown formatting
  - Semantic rewriting or stylistic improvements
  - Automatic heuristics beyond explicitly defined behavior
  - “Helpful” transformations not specified by fixtures

Agents must not expand scope beyond the defined transformations.


## Architectural Principles

Tool Granularity

- Each binary performs exactly one transformation
- Closely related but inverse operations (e.g. wrap vs. unwrap) must be separate tools
- Tools must be chainable via pipes

Binary Layout

- One binary per tool
- All tools live in a single monorepo
- Shared internal packages are allowed for:
  - Markdown parsing
  - Tokenization
  - Common I/O utilities

Do not create “umbrella” binaries.


## CLI Contract

Each tool must conform to the following interface:

Input / Output

- Accept input from:
  - STDIN
  - One or more file paths as arguments
- Write output to:
  - STDOUT by default
  - Optional explicit output file arguments (if provided)

If both STDIN and file arguments are provided, file arguments take precedence.

Exit Codes

- `0`: Successful execution (even if no changes were made)
- `>0`: Fatal error (invalid input, parse failure, I/O error)

Tools must not use exit codes to signal “no-op” results.

Determinism

- Output must be deterministic
- Given identical input, output must be byte-for-byte identical


## Transformation Rules

### Idempotency

All transformations must be idempotent:

```
T(T(input)) == T(input)
```

Agents must include fixtures that demonstrate idempotency.

### Isolation

- A tool may only perform its declared transformation
- It must not:
- Reformat unrelated Markdown
- Normalize whitespace outside its scope
- Reorder content unless explicitly required


## Fixtures & Testing

Fixture Style

- Use golden-file testing
- Each fixture consists of:
- input.md
- expected.md

Fixtures define behavior. Code must conform to fixtures, not vice versa.


### Coverage Expectations

Fixtures should cover:

- Edge cases
- Mixed Markdown constructs
- No-op scenarios
- Idempotency cases

Property-based testing is optional but welcome.


## Parsing Strategy

- Markdown AST usage is optional
- Full-document parsing is allowed and expected for some tools (e.g. reference-style links)
- Streaming behavior is not required

Agents should prefer correctness and clarity over premature optimization.


## Language & Build Constraints

- Implementation language: Go
- Deliverable: single static binary per tool
- Shared code must live in internal packages
- Run `gofmt -w` on all Go files after editing

External dependencies should be minimized but are not forbidden.


## Prohibited Behaviors

Agents must not:

- Add configuration files without explicit instruction
- Introduce global formatting opinions
- Perform multi-pass transformations unless required
- Guess user intent beyond fixtures
- Expand supported Markdown features implicitly

When in doubt: do less.


## Agent Guidance

When implementing a new tool:

1. Start from fixtures
2. Define the minimal transformation
3. Preserve everything else exactly
4. Verify idempotency
5. Keep the binary small and focused

Agents should treat this document as binding.
