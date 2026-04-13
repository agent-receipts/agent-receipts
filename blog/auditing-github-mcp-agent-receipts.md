# Every GitHub Action My AI Takes Is Now Cryptographically Signed

I've been building [Agent Receipts](https://agentreceipts.ai) — an open protocol for
cryptographically signed audit trails of AI agent actions. The idea is simple: when an
AI agent does something on your behalf, you should be able to prove what happened. Not
just logs. Proof.

This week I used it to audit itself.

---

## The problem with MCP tool calls

When you give Claude access to GitHub via the MCP server, it can create issues, push
files, open pull requests. It acts on your behalf. And when it's done, it tells you what
it did.

But that's a claim. Not evidence.

The logs are self-reported. There's no independent witness to what payload was sent,
which API was called, or what the server actually responded. In most cases that's fine.
But if you're operating agents in any context where accountability matters — a shared
codebase, a production system, a compliance environment — "the agent said so" isn't
enough.

Cryptographic receipts are the mechanism that closes this gap. A certifying proxy sits
between the MCP client and the server. Every tool call passes through it. Every call
gets a signed receipt. The receipt is hash-chained to the previous one, so the audit
trail is tamper-evident: you can't remove a receipt, reorder them, or insert a fake one
without breaking the chain.

---

## Routing the GitHub MCP server through the proxy

The Agent Receipts proxy is a transparent stdin/stdout wrapper. It doesn't modify the
GitHub MCP server or Claude Desktop. It just intercepts the JSON-RPC messages, signs a
receipt for each tool call, and forwards everything through.

The setup is three commands and a config change.

**Install the proxy:**

```bash
go install github.com/agent-receipts/ar/mcp-proxy/cmd/mcp-proxy@latest
```

**Generate a persistent signing key:**

```bash
mkdir -p ~/.agent-receipts
openssl genpkey -algorithm Ed25519 -out ~/.agent-receipts/github-proxy.pem
openssl pkey -in ~/.agent-receipts/github-proxy.pem -pubout \
  -out ~/.agent-receipts/github-proxy-pub.pem
```

**Update `claude_desktop_config.json`:**

```json
{
  "mcpServers": {
    "github-audited": {
      "command": "/Users/YOU/go/bin/mcp-proxy",
      "args": [
        "-name", "github",
        "-key", "/Users/YOU/.agent-receipts/github-proxy.pem",
        "-receipt-db", "/Users/YOU/.agent-receipts/receipts.db",
        "/opt/homebrew/bin/mcp-server-github"
      ],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "YOUR_TOKEN"
      }
    }
  }
}
```

Restart Claude Desktop. The GitHub MCP server now runs behind the proxy — every tool
call goes through it transparently.

---

## What a receipt looks like

After asking Claude to list open issues on the repo, I checked the receipt store:

```bash
mcp-proxy list -receipt-db ~/.agent-receipts/receipts.db
```

```
ID                                       ACTION   RISK   STATUS    TIMESTAMP
---
urn:receipt:fc72deca-e3b7-49c8-a2c5-8…  read     low    success   2026-04-12T06:32:02Z
urn:receipt:4f123102-8c0a-4c46-abf8-0…  read     low    success   2026-04-12T06:32:07Z

2 receipts
```

`Action: read`, `Risk: low` — the proxy recognised `list_issues` as a read operation
and scored it accordingly.

Each row in that list is a W3C Verifiable Credential stored on disk. Here's what one
actually looks like:

```json
{
  "@context": [
    "https://www.w3.org/ns/credentials/v2",
    "https://agentreceipts.ai/context/v1"
  ],
  "id": "urn:receipt:fc72deca-e3b7-49c8-a2c5-8b3f1a2c9d4e",
  "type": ["VerifiableCredential", "AgentReceipt"],
  "version": "0.1.0",
  "issuer": {
    "id": "did:agent:mcp-proxy"
  },
  "issuanceDate": "2026-04-12T06:32:02Z",
  "credentialSubject": {
    "principal": {
      "id": "did:user:unknown"
    },
    "action": {
      "id": "act_a98cd38a-6bf6-4c14-86f1-cebebf0def83",
      "type": "read",
      "tool_name": "list_issues",
      "risk_level": "low",
      "target": { "system": "github" },
      "parameters_hash": "sha256:3846c4a1f3b2e9d7c0581a4f6e2b8d3c9a7f1e4b",
      "timestamp": "2026-04-12T06:32:02Z"
    },
    "outcome": {
      "status": "success"
    },
    "chain": {
      "sequence": 1,
      "chain_id": "fc72deca-e3b7-49c8-a2c5-8b3f1a2c9d4e",
      "previous_receipt_hash": null
    }
  },
  "proof": {
    "type": "Ed25519Signature2020",
    "created": "2026-04-12T06:32:02Z",
    "verificationMethod": "did:agent:mcp-proxy#key-1",
    "proofPurpose": "assertionMethod",
    "proofValue": "uX3bbrW7YoSWlqZzBvAr5iUjDOA05Na0aPb1efc72de..."
  }
}
```

The `parameters_hash` is a SHA-256 of the RFC 8785 canonical JSON of the tool call
arguments — the actual payload sent to the GitHub API, not a summary. The `proof` is
an Ed25519 signature over the canonical receipt. The `chain` links each receipt to the
previous one by hash, making the sequence tamper-evident.

Then I verified the chain:

```bash
mcp-proxy verify -key ~/.agent-receipts/github-proxy-pub.pem \
  -receipt-db ~/.agent-receipts/receipts.db \
  fc72deca-e3b7-49c8-a2c5-8b3f1a2c9d4e
```

```
Chain fc72deca-e3b7-49c8-a2c5-8b3f1a2c9d4e: VALID (2 receipts)
```

Every tool call adds another receipt. The chain grows with the session — tamper-evident
from the first call to the last.

---

## The meta moment

Here's the part I enjoyed: once writes were working, I used the audited connection to
file bug reports discovered during the session itself.

The proxy wasn't storing the tool name in receipts at all — every call showed
`Action: unknown`. The classifier code was correct, but the tool name was never wired
through from interception to storage. And the CLI reference docs were showing
double-dash flags (`--db`) when the binary uses single-dash Go flags (`-db`).

So I filed both bugs via the proxy:

- [#109 — tool name not stored in receipt, action.type always unknown](https://github.com/agent-receipts/ar/issues/109) — the tool name was never wired from JSON-RPC interception through to the classifier or the stored receipt, fixed in v0.3.3 of the proxy
- [#101 — Docs: CLI reference shows double-dash flags but binary uses single-dash](https://github.com/agent-receipts/ar/issues/101)

Those GitHub API calls — `create_issue`, `update_issue` — have signed receipts from
the GitHub MCP server running through the proxy. The bug reports about Agent Receipts
were filed using Agent Receipts, via the GitHub MCP server, and the act of filing them
is itself receipted.

That's the dogfooding loop closing.

---

## What the chain gives you

Each receipt is an Ed25519-signed W3C Verifiable Credential. The chain property means:

- **Tamper-evidence**: remove or reorder a receipt and the chain verification fails
- **Gap detection**: a missing sequence number is immediately visible
- **Independent verification**: anyone with the public key can verify the chain — no
  trust in the proxy required
- **Cross-session continuity**: because the key is persistent, receipts from different
  sessions can be verified against the same public key

The proxy also handles sensitive data automatically — GitHub PATs, API keys, and other
secrets matching known patterns are redacted before storage. Your token isn't in the
audit log.

---

## What's still rough

This is early days — proxy v0.3.3, SDKs v0.3.0. A few things I hit during the walkthrough:

**Placeholder DIDs.** The issuer shows as `did:agent:mcp-proxy` and principal as
`did:user:unknown`. These are placeholders — the DID method strategy is being worked
out in [ADR-0007](https://github.com/agent-receipts/ar/issues/46). For now they're
labels, not resolvable identifiers.

**Fine-grained PATs don't work for org write access.** GitHub's org-level policy for
fine-grained PATs creates friction even when permissions look correct. Use a classic
PAT with `repo` scope for org-owned repos for now.

---

## Try it

The proxy is open source, ships as a single Go binary, and wraps any MCP server —
not just GitHub.

```bash
go install github.com/agent-receipts/ar/mcp-proxy/cmd/mcp-proxy@latest
```

Full walkthrough: [Auditing Your GitHub MCP Server with Agent Receipts](https://agentreceipts.ai/mcp-proxy/overview/)

The spec, SDKs, and proxy are all in the monorepo at
[github.com/agent-receipts/ar](https://github.com/agent-receipts/ar).

If you're working on MCP tooling, agent governance, or verifiable credentials and want
to review the protocol design — particularly the forward-looking ADRs on DID methods,
key rotation, and UCAN delegation — I'd welcome the input.
