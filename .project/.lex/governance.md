# Specification governance

Status: documentation governance for the canary release.

## Authority and requirements

The [canonical index](README.md) identifies document owners and precedence.
Normative requirements describe conforming behavior; implementation status
is recorded in the [plan](../.plan/progress.md), not asserted by the text. Examples and historical research are informative
even when they contain imperative language.

Uppercase MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY use the requirement levels
of [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and
[RFC 8174](https://www.rfc-editor.org/rfc/rfc8174). Lowercase uses are ordinary
prose. This convention does not make LeX an IETF standard.

## Change and release lifecycle

The lifecycle is canary -> stable.

The current release is **canary**: generic envelope `0.2` and legacy envelope
`0.1`. Canary promises:

- request, response, and bundle fields and their meaning are fixed within an
  envelope version; changes are additive and disclosed in capabilities;
- a breaking change to a message, hash rule, or answer meaning gets a new
  envelope version, and the previous verifier stays available for replay;
- problem `reason` codes and HTTP status mapping are stable.

Canary does not claim conformance certification, calibration, or an achieved
SLO. Stable follows after several independent products use LeX in production
and the open gates in [progress.md](../.plan/progress.md) close.

For semantic changes, record the problem, affected contract, alternatives,
compatibility impact, security implications, and required proof in a reviewed
change proposal. Record maintainer acceptance before release; model output
cannot substitute for acceptance. Update canonical text and dependent links
together. Preserve research captures.

Stable requires fixed normative documents, immutable schema identifiers,
testable requirement identifiers, a requirement-to-fixture map, the
[proof gates](analysis.md), a versioned conformance report, and recorded
limitations.

Distinguish editorial corrections from behavior changes.
Changes to accepted messages, hashes, verdict meaning, or required behavior
need compatibility review and an explicit version decision. Optional fields
are not automatically compatible: validators and hash rules may reject them.
Never change a published artifact under the same immutable identifier.

## Open decisions

Track unresolved decisions in [analysis.md](analysis.md) until specified and
tested. Do not invent endpoints, wire fields, extension behavior, or release
guarantees in examples. Keep provider details out of core verdict semantics.
