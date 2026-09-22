# LeX specification

Status: canary release, envelope `0.2`. Compatibility promises are in
[governance.md](governance.md).

Read in order:

1. [concept.md](concept.md): purpose, primitives, and boundaries (informative).
2. [protocol.md](protocol.md): operation, invariants, entities, lifecycle, replay (normative).
3. [checks.md](checks.md): request checks, finding codes, problem reasons, HTTP status (normative).
4. [scope.md](scope.md): what is built, open, and refused (normative boundary).
5. [client-integration.md](client-integration.md): calling LeX from applications and agents.
6. [governance.md](governance.md): requirement keywords, release lifecycle, change control.

`protocol.md` owns shared semantics; `checks.md` refines it. Every MUST line in
those two files is mapped to a test by `internal/conformance/obligations_test.go`.
The wire contract is `internal/wire/schema/`. Examples and research never
override this text.
