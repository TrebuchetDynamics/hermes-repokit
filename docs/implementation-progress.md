# Implementation progress — 2026-09-27

The final user directive is the target; v1 is **not complete**.

## Completed implementation foundations

- Separate plan correction commit `a90381b`: Go host, upstream Nerve/Laya, native OpenViking, native review first, automatic memory extraction accepted.
- Canonical Go module and typed version-bound qualification contracts.
- Canonical repository identity, exact public names, path-hashed Compose project; filesystem/PATH/root Compose checks. CLI additionally inspects Git tracking and Docker names.
- Digest-pinned readable Compose renderer, private atomic directory publication with Linux `RENAME_NOREPLACE`, fsync, kernel locking and conservative reruns.
- Standalone POSIX launcher with captured Docker context, absolute Compose path, selector clearing, exact argv/streams/exit behavior and terminal selection.
- Native setup delegation and context-correct stopped-service instruction.
- Read-only JSON `plan` and metadata-only foundation `verify`; no receipt trust or native DB calls.
- Hermes-only CLI `install`, with locked inventory recheck, context-preserving safe reruns, native config preservation and explicit ordinary Compose startup handoff.
- Exact plugin caution-admission predicate; no forced install.
- Offline unit/acceptance coverage including hostile argv, TTY matrix, SIGINT exit, child-held lock, stale lock file, owner edits, missing/corrupt receipts, failed preparation, concurrent publication, interrupted/partial-state refusal and process-group cleanup.
- Every Go package has tests, including the executable entrypoint.

## Actual runtime evidence

Credential-free `TestDockerFoundation` passed: generated Compose parsed; a new
official Hermes container ran from an unrelated disposable repository; native
exec and Kanban initialization worked; raw Compose restart preserved the board.
Cleanup used ordinary Compose down, without `-v`.

Separately observed native profile creation/description and setup help on the
pinned image. Superpowers native installation failed closed at CAUTION with
229 findings for the exact recorded SHA. The official OpenViking image starts
its missing-config handoff; native init reports persistent workspace
`/app/.openviking/data` and cancels without required credentials. No inference,
actual memory write/recall or cross-repository denial was claimed.

Fresh independent code review identified five important issues. Regression
checks reproduced and fixed: no-clobber publication, incomplete reruns,
descendant cleanup, verify Docker context, and setup recovery context/selectors.
`Publish.prepare` is explicitly restricted to immutable image retrieval;
future native initializer subprocesses need inherited locking before wiring.

## Outstanding, in order

1. Foundation complete: real CLI plan/install/verify/rerun and source/binary removal were exercised with the pinned Hermes image. Offline tests also exercise setup delegation and receipt removal. No optional integration is enabled by this result.
2. Next, admit exact Superpowers SHA/findings after explicit user approval and qualify actual native loading. Never promote scanner success from mocks.
3. Persist safe default config and create/preserve the engineering profiles with qualified native commands/toolsets; keep credentials isolated and dispatch/decomposition off.
4. Qualify Nerve catalog revision/scanner/profile loading and its Laya sidecar transport/checkpoint. Current upstream rejects `http://laya:8765`; choose and qualify a supported transport, do not invent another protocol.
5. Finish official OpenViking scaffold/native provider configuration and operator handoff; actual model setup needs provider/model/budget selection and private native credential entry.
6. Prove distinct same-card builder/reviewer actors. Add native policy only if an enforcement gap is demonstrated.
7. Complete read-only component-specific evidence/probes, installation failure/recovery matrix and full authorized integration matrix.
8. Perform the full removal-first release gate, then source dogfood, then release-qualified builds/checksums on both supported architectures.

## Scope and evidence rulings

- Reused the existing `qualification` package instead of duplicating `qualify`; no external dependencies means no fabricated `go.sum`.
- Native readable child files beneath a 0700 state root are allowed; native Hermes itself creates 0755 profile directories. Group/world writable native children still refuse.
- Metadata observations can proceed while plugin/model admission is blocked. No later runtime integration is enabled based on guessed commands.
- Development binaries are not a v1 release. The local Go 1.26.1 toolchain needs patched-release qualification before shipping.
- Ruling: the latest review moves the Hermes-only foundation ahead of optional integrations. `install` publishes files only; startup belongs to ordinary Compose, and native configuration/credentials belong to native setup. The first milestone does not depend on plugin scanner approval or full v1 model evidence.
- Concurrent edits to TODO, the bootstrap spec/plan and native-contract/upstream-Nerve notes were preserved separately from implementation commits.

## Foundation closeout validation

Fresh `go test ./... -count=1`, race tests, vet, formatting, diff checks and the
real Docker foundation fixture passed. Local links in the changed documentation
were checked. An independent review reproduced missing Git-ignore protection on
safe reruns and verification; regression tests failed before the fix and passed
after adding shared read-only index/ignore checks. Removed ignore files, ignore
exceptions and force-tracked native state now refuse without rewriting files.
The reviewer checked the fix and found no additional actionable issue.
