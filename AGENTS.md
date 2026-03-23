## Code Exploration Policy

Always use jCodemunch-MCP tools for code navigation. Never fall back to Read, Grep, Glob, or Bash for code exploration.

**Start any session:**
1. `resolve_repo { "path": "." }` — confirm the project is indexed. If not: `index_folder { "path": "." }`
2. `suggest_queries` — when the repo is unfamiliar

**Finding code:**
- symbol by name → `search_symbols` (add `kind=`, `language=`, `file_pattern=` to narrow)
- string, comment, config value → `search_text` (supports regex, `context_lines`)
- database columns (dbt/SQLMesh) → `search_columns`

**Reading code:**
- before opening any file → `get_file_outline` first
- one symbol → `get_symbol`; multiple → `get_symbols` (fewer round-trips)
- symbol + its imports → `get_context_bundle`
- specific line range only → `get_file_content` (last resort)

**Repo structure:**
- `get_repo_outline` → dirs, languages, symbol counts
- `get_file_tree` → file layout, filter with `path_prefix`

**Relationships & impact:**
- what imports this file → `find_importers`
- where is this name used → `find_references`
- is this dead code → `check_references`
- file dependency graph → `get_dependency_graph`
- what breaks if I change X → `get_blast_radius`
- class hierarchy → `get_class_hierarchy`
- related symbols → `get_related_symbols`
- diff two snapshots → `get_symbol_diff`

**After editing a file:** `index_file { "path": "/abs/path/to/file" }` to keep the index fresh.

## Context7 MCP Policy

Use Context7 MCP for external library/framework/API documentation, version-specific usage, setup, configuration, and code generation that depends on current docs. Do not use it for repository code exploration; use jCodemunch for local code.

**When to use Context7:**
- library or framework API questions
- dependency setup or configuration steps
- generating code that depends on third-party docs
- verifying current usage against upstream docs or deprecations

**How to use Context7 correctly:**
1. If the exact Context7 library ID is unknown, call `resolve-library-id` first.
2. Then call `query-docs` with the resolved library ID and a focused query.
3. If the exact library ID is already known, skip resolution and use the ID directly.
4. Include the library version in the query when the user specifies one or when version differences matter.
5. Prefer official/vendor libraries and docs when multiple matches exist.

**Prompting guidance:**
- If relevant, explicitly add `use context7`.
- If the library is known, include its exact ID in the prompt, for example: `use library /supabase/supabase`.
- Keep Context7 queries narrow and task-specific; ask for the exact API, feature, or configuration you need.

**Safety and privacy:**
- Never send secrets, API keys, passwords, personal data, or proprietary code in Context7 queries.
- Use Context7 only for external documentation retrieval; keep full local code/context in the agent unless it is essential to the documentation question.
