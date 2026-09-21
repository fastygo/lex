# Specification governance

Status: documentation governance for the v0.1 working draft.

## Authority and requirements

The [canonical index](README.md) identifies document owners and precedence.
Normative draft requirements describe intended conforming behavior; they do
not assert implementation. Examples and historical research are informative
even when they contain imperative language.

Uppercase MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY use the requirement levels
of [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) and
[RFC 8174](https://www.rfc-editor.org/rfc/rfc8174). Lowercase uses are ordinary
prose. This convention does not make LeX an IETF standard.

## Change and release lifecycle

The lifecycle is working draft -> release candidate -> released specification.
The current version is a working draft. A `0.1` label in a document or fixture
does not establish a compatibility promise or conformance certification.

For semantic changes, record the problem, affected contract, alternatives,
compatibility impact, security implications, and required proof in a reviewed
change proposal. Record maintainer acceptance before release; model output
cannot substitute for acceptance. Update canonical text and dependent links
together. Preserve research captures.

A release candidate needs fixed normative documents, immutable schema
identifiers, testable requirement identifiers, and a requirement-to-fixture map.
Release requires the [proof gates](analysis.md), a versioned conformance report,
and recorded limitations.

After release, distinguish editorial corrections from behavior changes.
Changes to accepted messages, hashes, verdict meaning, or required behavior
need compatibility review and an explicit version decision. Optional fields
are not automatically compatible: validators and hash rules may reject them.
Never change a published artifact under the same immutable identifier.

## Open decisions

Track unresolved decisions in [analysis.md](analysis.md) until specified and
tested. Do not invent endpoints, wire fields, extension behavior, or release
guarantees in examples. Keep provider details out of core verdict semantics.
