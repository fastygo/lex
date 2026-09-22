# Governance

Status: change control for the canary release.

## Requirement keywords

Uppercase MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY follow
[RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and
[RFC 8174](https://www.rfc-editor.org/rfc/rfc8174). This does not make LeX an
IETF standard. Implementation status lives in
[progress.md](../.plan/progress.md), not in normative text.

## Release lifecycle

The lifecycle is canary -> stable. The current release is **canary**,
envelope `0.2`. Canary promises:

- request, response, and bundle fields and their meaning are fixed within an
  envelope version; changes are additive and disclosed in capabilities;
- a breaking change to a message, hash rule, or answer meaning gets a new
  envelope version, and the previous verifier stays available for replay;
- problem `reason` codes, finding codes, and the HTTP status mapping are stable.

Canary does not claim conformance certification, calibration, or a measured
SLO. Stable follows after several independent products use LeX in production
and the open gates in [progress.md](../.plan/progress.md) close.

## Change control

- Change the owning document and its test together; never keep a parallel copy.
- A new MUST line needs a test mapped in `internal/conformance/obligations_test.go`.
- A durable boundary change needs an entry in
  [decisions.md](../.plan/decisions.md).
- Distinguish editorial corrections from behavior changes. Optional fields are
  not automatically compatible: strict validators and hashes may reject them.
- Never change a published artifact under the same identifier.
- Keep provider details out of protocol objects and answer semantics.

## References

- [RFC 8785](https://www.rfc-editor.org/rfc/rfc8785) JSON Canonicalization Scheme
- [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) Problem Details for HTTP APIs
- [JSON Schema 2020-12](https://json-schema.org/draft/2020-12)
- [OpenAPI 3.1.1](https://spec.openapis.org/oas/v3.1.1)
