# ADR-0007: Request RAM and caller-owned replay

Status: proposed. Date: 2026-09-21.
Scope constraints in [the plan](../README.md) are fixed; implementation details
require acceptance evidence. Owner: LeX maintainers. No implementation claimed.

## Context

Canonical raw-answer preservation is required, but the service has no persistent store.

## Decision

Retain exact answers and frozen inputs in request RAM and return a complete bounded replay bundle. A later replay supplies its own bundle; it never retrieves or calls a model. Keep only immutable configuration and safe HTTP transport pools globally. No artifact cache, disk writes, database, or external storage.

## Consequences and alternatives

A successful response exports reproducibility material, not a durable storage receipt. A lost response is unrecoverable. The caller chooses whether to retain it. Replay proves computation under supplied pinned inputs, not historical authenticity or present authorization. Personal data must be excluded or minimized according to the profile before evaluation.

## Acceptance evidence

Network-disabled replay on a fresh instance; identical findings/verdict; no filesystem writes; no cross-request/project state; complete bundle round-trip inside both request and response caps; intentional response loss documented.

## Links

[ADR register](README.md) | [Delivery gates](../delivery.md) | [Conformance](../conformance.md)
