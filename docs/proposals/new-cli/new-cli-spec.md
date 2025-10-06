# Design Specification: A Simpler, Ergonomic CLI Query Syntax

**Status:** Proposal
**Date:** 2025-10-05

### 1. Introduction & Goal

The current NanoStore CLI maps directly to the Go API, providing a powerful but complex interface. For the vast majority of command-line interactions, the primary task is simple data retrieval with basic filtering.

The goal of this proposal is to design an ergonomic and intuitive CLI syntax specifically for these common query cases. This CLI will complement, not replace, the existing advanced client. It is optimized for speed, readability, and ease of use.

### 2. Core Design Philosophy: Domain-Specific Ergonomics

The CLI will be treated as a **Domain-Specific Language (DSL)** for querying. The core principles are:

- **Conciseness:** The most common actions should require the fewest keystrokes.
- **Readability:** Queries should read like natural language where possible.
- **Discoverability:** The tool should be easy to learn and use.

### 3. Syntax Specification

#### 3.1. Basic Equality Filtering

For simple equality checks, the flag is the field name itself.

```bash
# SQL: SELECT * FROM documents WHERE user = "alice" AND status = "active";
$ nanostore get --user="alice" --status="active"
```

#### 3.2. Advanced Operators

For comparisons other than equality, a double-underscore (`__`) suffix is used. This is an unambiguous delimiter that avoids conflicts with field names that may contain dashes.

**Syntax:** `--<field-name>__<operator>=<value>`

**Proposed Operators:**

| Operator     | Description              | Example                           |
| :----------- | :----------------------- | :-------------------------------- |
| `gt`         | Greater Than             | `--id__gt=100`                    |
| `gte`        | Greater Than or Equal To | `--priority__gte=5`               |
| `lt`         | Less Than                | `--stock__lt=10`                  |
| `lte`        | Less Than or Equal To    | `--version__lte=2`                |
| `ne`         | Not Equal To             | `--status__ne=archived`           |
| `contains`   | Substring contains       | `--title__contains="cli syntax"`  |
| `in`         | Value is in a list       | `--tags__in="dev,ops"`            |
| `startswith` | String starts with       | `--hostname__startswith="prod-"`  |
| `endswith`   | String ends with         | `--file__endswith=".md"`          |

#### 3.3. Logical Conditions: `AND` & `OR`

- **Implicit `AND`:** Multiple filter flags are joined with a logical `AND` by default.
- **Explicit `OR` (and `AND`):** Special infix flags `--or` and `--and` are used to combine conditions.
  - **Precedence Rule:** The query is evaluated with **strict left-to-right precedence.** This is a crucial simplification for the CLI and must be clearly documented.

```bash
# (label CONTAINS "buy" OR label CONTAINS "read") AND id > 4
$ nanostore get --label__contains="buy" --or --label__contains="read" --id__gt=4

# priority >= 5 AND (status = "new" OR user = "guest")
$ nanostore get --priority__gte=5 --and --status="new" --or --user="guest"
```

#### 3.4. Command Flags vs. Filter Flags

To prevent collisions between the CLI's operational flags and data field names, all built-in command flags will be namespaced with an `x-` prefix.

- `--x-db=...`
- `--x-format=json`
- `--x-verbose`

This creates a clear, unambiguous separation between "how the command runs" and "what data the command finds."

### 4. Testing Strategy

- The focus will be on **unit testing** the pre-parser's transformation of arguments (`[]string`) into a well-defined `Query` object.
- No integration tests that hit the NanoStore core or database will be part of this work.
- Standard flags like `--help` and `-h` will be passed through to the underlying command framework (Cobra) to ensure existing help behavior is preserved.

### 5. Out of Scope for Initial Implementation

- **Dynamic, Schema-Aware Help:** A system where `nanostore get --help` connects to the datastore to generate a list of available filters is a future enhancement, not part of this initial scope.
- **Complex, Nested Logic:** The left-to-right precedence rule means that deeply nested queries like `(A AND B) OR (C AND D)` are not supported. Users needing this level of complexity should be directed to use the full-featured API.
