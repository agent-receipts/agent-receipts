# ADR-0003: W3C Verifiable Credentials as the Receipt Envelope

## Status

Accepted

## Context

The Agent Receipts protocol needs a structured envelope format for receipts that supports cryptographic proofs, issuer identification, and interoperability with existing trust ecosystems. The format must be extensible enough to carry domain-specific receipt content (action details, chain linkage, authorization, delegation) while providing a clear separation between envelope metadata and receipt payload. It must also support the receipt chain model, where each receipt references its predecessor via a hash link.

We evaluated the following alternatives:

- **W3C Verifiable Credentials Data Model 2.0:** Industry standard for verifiable claims. Provides a well-defined structure for `issuer`, `credentialSubject`, and `proof`, with broad tooling and growing adoption across identity, supply chain, and compliance ecosystems. DID-based issuer identification aligns with decentralized agent identity.
- **Custom JSON envelope:** Maximum flexibility to design fields specific to agent receipts, but no existing ecosystem, no shared tooling, and no interoperability with trust infrastructure. Every consumer would need custom parsing logic. The cost of designing, documenting, and defending a bespoke envelope format is high for a new protocol seeking adoption.
- **JSON Web Signatures (JWS/JWT):** Compact and widely deployed for authentication tokens. However, JWT's flat claims model is a poor fit for the structured, nested content in receipts (action, outcome, chain, delegation). Extending JWT claims for rich structured data requires non-standard usage that negates interoperability benefits. JWS detached payloads could work but add complexity without ecosystem alignment.
- **CBOR/COSE (RFC 9052):** Efficient binary format with built-in signing (COSE). Attractive for bandwidth-constrained environments, but loses JSON readability, browser-native parsing, and developer tooling. Agent receipts are audit records meant to be human-inspectable; a binary format works against this goal. CBOR also has a smaller tooling ecosystem for credential verification workflows.

Related: #44, #20 (parent issue).

## Decision

Use the W3C Verifiable Credentials Data Model 2.0 as the envelope format for all agent receipts. Receipts are W3C VCs with `type: ["VerifiableCredential", "AgentReceipt"]` and a required `@context` array beginning with the W3C VC v2 context URI (`https://www.w3.org/ns/credentials/v2`) followed by the Agent Receipts context URI (`https://agentreceipts.ai/context/v1`).

Key structural choices:

- **`issuer`** identifies the agent instance via a DID (e.g., `did:agent:...`), with optional fields for agent type, name, operator, model, and session identifier.
- **`credentialSubject`** carries the entire receipt payload: principal, action, intent, outcome, authorization, delegation, and chain linkage. This keeps domain-specific content cleanly separated from the VC envelope.
- **`proof`** follows the W3C VC Data Integrity structure with `type`, `created`, `verificationMethod`, `proofPurpose`, and `proofValue`. The proof type is `Ed25519Signature2020` (see ADR-0001).
- **`validFrom`** serves as the issuance timestamp per VC Data Model 2.0 naming.
- **`version`** is a protocol extension field (`"0.1.0"`) not defined by the VC Data Model. VC tooling that validates strictly against the VC schema should ignore unrecognized top-level fields.

The protocol uses the W3C VC JSON shape but does not require a VC library dependency. Receipts may be constructed with plain JSON serialization (spec 10.1). No JSON-LD processing is required — the `@context` field is included for VC ecosystem compatibility but is not dereferenced at runtime.

### Deviations from W3C defaults

The protocol intentionally deviates from certain W3C Data Integrity defaults to simplify implementation:

1. **Canonicalization:** The signing input uses RFC 8785 (JSON Canonicalization Scheme) rather than JSON-LD canonicalization (URDNA2015). This avoids a JSON-LD processing dependency while still producing deterministic byte sequences for signing. The W3C Data Integrity spec references RFC 8785 as an option; we adopt it as the sole method.
2. **Signing algorithm:** The signer computes the Ed25519 signature directly over the RFC 8785 canonical JSON of the receipt with the `proof` field removed. This is simpler than the full W3C Data Integrity signing algorithm, which constructs a separate verification hash from document and proof options (spec 10.2).
3. **`proofValue` encoding:** The JSON schema specifies multibase `z`-prefixed base58btc encoding for `proofValue`, consistent with the W3C Data Integrity default. Implementations must use this encoding. (Note: ADR-0001 references base64url with `u` prefix in the encoding section; the schema is authoritative — see spec 10.2 for the documented deviation boundary.)

These deviations are documented in spec 10.2. Implementations that need full W3C Data Integrity compatibility should use the complete algorithm and note this in their conformance documentation.

## Security Considerations

### Subset compliance

Agent Receipts use a subset of the W3C VC Data Model. Features not used include: `credentialStatus`, `credentialSchema` (as a VC-level field), `evidence`, `termsOfUse`, `refreshService`, and `holder`. The `type` array is fixed to `["VerifiableCredential", "AgentReceipt"]` — no additional VC types are supported. Implementers expecting full W3C VC compliance should be aware of these boundaries. The subset used is documented in the JSON schema at `spec/schema/agent-receipt.schema.json`, which uses `additionalProperties: false` to reject unrecognized fields.

### Proof structure integrity

The `proof` field is excluded from the signing input (the canonical JSON is computed with `proof` removed). This follows the W3C Data Integrity pattern where the proof is not self-signed. A receipt's integrity depends on the proof being verified against the receipt body — any modification to non-proof fields after signing will invalidate the signature.

### verificationMethod resolution

The `proof.verificationMethod` field contains a DID URL (e.g., `did:agent:example#key-1`) identifying the signing key. DID resolution — the process of resolving this URL to an actual public key — is not specified in v0.1 of the protocol (see spec 9.6). Verifiers must currently obtain public keys through out-of-band means. This is a known gap: without a defined resolution mechanism, verifiers cannot independently discover signing keys, and the `verificationMethod` value is effectively an opaque identifier rather than a resolvable reference. A companion specification or future protocol version must address DID method requirements and key discovery.

### @context injection

The `@context` array is currently not processed as JSON-LD — it is included for VC ecosystem compatibility only. If JSON-LD processing is ever added, `@context` manipulation becomes a significant attack vector: a malicious `@context` URI could redefine term semantics, causing fields to be interpreted differently than intended. The schema constrains the first two context entries to fixed values, but additional context URIs are permitted. Implementations must not dereference or process `@context` URIs without explicit security review.

### Envelope field manipulation

The separation of envelope (`issuer`, `validFrom`, `version`) from payload (`credentialSubject`) means both layers are covered by the signature. However, the `@context` and `type` fields are also signed, and any modification to these fields will break verification. Implementations should verify the full receipt structure, not just the `credentialSubject`, when checking signature validity.

## Known Risks

- **Subset confusion.** Implementers familiar with the W3C VC ecosystem may expect features (credential status, holder binding, JSON-LD processing) that are not present. This may cause integration friction or false assumptions about protocol capabilities.
- **W3C VC spec evolution.** The VC Data Model is actively maintained. Future versions may introduce breaking changes to field names, required fields, or proof structure. The protocol's `version` field and JSON schema provide a buffer, but schema evolution will be needed to track upstream changes.
- **JSON-LD overhead without JSON-LD processing.** Including `@context` without processing it adds payload size and conceptual complexity. Developers unfamiliar with JSON-LD may find the field confusing or attempt to process it, introducing unintended behavior.
- **DID ecosystem coupling.** The `issuer.id` and `proof.verificationMethod` fields assume DID-based identifiers. While no specific DID method is required, this ties the protocol to the DID ecosystem conceptually. If DID adoption stalls or fragments, the protocol's identity model may need revision.

## Consequences

- Receipt schema follows the W3C VC Data Model 2.0 structure. All receipts have `@context`, `type`, `issuer`, `validFrom`, `credentialSubject`, and `proof` at the top level.
- All SDKs (Go, Python, TypeScript) must implement the VC envelope structure and proof format per W3C VC Data Integrity, with the simplifications documented in spec 10.2.
- Deviations from W3C defaults (RFC 8785 canonicalization, simplified signing input) must be explicitly documented in SDK conformance statements and any interoperability guides.
- Interoperability with W3C VC tooling is possible but not guaranteed without full compliance. Receipts can be parsed by generic VC libraries, but verification may fail if the library expects the full Data Integrity signing algorithm.
- Future W3C VC spec updates may require schema evolution. The `version` field provides a migration path.
- The `additionalProperties: false` constraint in the JSON schema means the envelope is not freely extensible. Adding new top-level fields requires a schema version bump.
- Receipt examples and test vectors across all SDKs serve as the normative reference for the VC envelope structure, ensuring cross-language consistency.
