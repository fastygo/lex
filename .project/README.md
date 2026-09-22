# LeX project documentation

Status: canary release (generic envelope `0.2`, legacy `0.1`); no stable
release or conformance certification is claimed.

## Directory ownership

- [Canonical specification](.lex/README.md): protocol contracts and explicitly marked planning documents.
- [Implementation plan](.plan/README.md): Go REST, Framework + Context, Vercel, RAM-only ADRs, SLOs, and standards acceptance.
- [Architecture guidance](.vsa/README.md): vertical slices and ICOM; subordinate to the protocol.
- [Research](.jev/README.md): non-normative manual inputs, captured responses, and historical notes.
- [Assets](.assets/README.md): informative illustrations.

Start with the [canonical reading order](.lex/README.md).

Edit the owning canonical document when behavior changes; link instead of
maintaining parallel specifications. Preserve research captures. Examples do
not prove implementation, calibration, interoperability, or safe execution.

Human-readable content here is English. Optional: `npm run check:english`.
