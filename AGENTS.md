# AGENTS.md

Monorepo for the Agent Receipts protocol — cryptographically signed audit trails for AI agent actions. Contains the protocol spec, SDKs in three languages, an MCP proxy, and the documentation site.

## Monorepo layout

```
spec/          # Protocol specification and JSON schemas
sdk/go/        # Go SDK (receipt, store, taxonomy)
sdk/ts/        # TypeScript SDK (@agent-receipts/sdk-ts)
sdk/py/        # Python SDK (agent-receipts)
mcp-proxy/     # MCP STDIO proxy with audit, policy, and receipts (Go)
site/          # Documentation site (Astro)
cross-sdk-tests/  # Cross-language receipt verification tests
```

Each subdirectory has its own AGENTS.md with project-specific details.

## Quick reference

| Component | Language | Test command | Build command |
|-----------|----------|-------------|---------------|
| sdk/go | Go | `go test ./...` | `go build ./...` |
| sdk/ts | TypeScript | `pnpm test` | `pnpm build` |
| sdk/py | Python | `uv run pytest` | `uv build` |
| mcp-proxy | Go | `go test ./...` | `go build ./cmd/mcp-proxy` |
| site | TypeScript | — | `pnpm build` |
| spec | — | — | JSON schema validation |

## Conventions

- All changes go through pull requests — never push directly to main
- CI is path-filtered: changes to `sdk/go/` only trigger Go SDK CI
- mcp-proxy CI also triggers on `sdk/go/` changes (dependency)
- Site deploys on `site/**` or `spec/**` changes
- Go modules use a `replace` directive for local development (mcp-proxy → sdk/go)
- Run language-specific linters before committing (go vet, biome, ruff)

## Dependencies

```
spec (protocol definition)
  ↓
sdk/go ← mcp-proxy (Go replace directive for local dev)
sdk/ts
sdk/py
```

SDKs are independent implementations of the same spec. They do not depend on each other but must produce compatible receipts (same canonical JSON, same signature encoding, same hash format).
