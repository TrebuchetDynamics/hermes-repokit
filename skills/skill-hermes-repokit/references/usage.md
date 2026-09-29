# Use and verify an installed repository

## Native commands

Use the verified host command, or the absolute generated launcher when there
is no PATH link. For a repository whose planned launcher is `hermes-design-notes`:

```sh
hermes-design-notes                         # native chat, explicitly default
hermes-design-notes -p researcher           # another existing native profile
hermes-design-notes profile list
hermes-design-notes kanban list
hermes-design-notes plugins list
hermes-design-notes gateway status
```

No-argument launch selects `default`; explicit arguments pass through unchanged.
`hermes-design-notes setup` is native Hermes setup only. It does not run RepoKit's
team or OpenViking sequence. Use the bootstrapper from the target
root when those steps are needed. Native command syntax is version-sensitive:
inspect scoped `--help` or matching upstream documentation before dispatch,
profile mutations, or memory commands. Do not invent RepoKit `chat`, `start`,
`run`, or `stop` subcommands.

## Runtime recovery

Read the recognized launcher for its captured Docker context and absolute
Compose file; refuse unknown routing. Use ordinary Compose with those selectors,
`--env-file /dev/null`, and the same interfering-variable cleanup as the
generated command. Native state stays in `.hermes`; do not delete volumes,
profiles, credentials, or board state as a recovery shortcut.

If only a service is stopped, start that service in the existing project.
Recreate the single Hermes service using generated pins/build inputs and inspect
health afterward. Embedded OpenViking's configuration/data remain in their persistent
mount. An unconfigured service is pending private setup, not a reason to invent
embedding models or bypass authentication.

If core execution works and only OpenViking is unhealthy, keep the engineering
team working and report memory degraded/pending. Use the selected revision's native
memory-only recovery when authorized; avoid a whole-runtime restart solely to
clear memory status. If that revision couples memory recovery to dispatcher
suspension, report the version gap before attempting it. Durable repairs use
RepoKit source or native Hermes configuration, never hand-edited generated files
or `.hermes` artifacts to force a passing result. See [version gaps and
migration](installation.md).

The bootstrap binary is unnecessary for chat, native commands, and Compose
recovery. If setup or reconciliation is needed after its removal, acquire a
compatible bootstrapper as described in [installation](installation.md), without
replacing runtime state. Installation does not automatically run a gateway.

## Team and Kanban work

Talk to `default`; it coordinates `researcher`, `planner`, `executor`, `reviewer`,
and `steward`. Use existing profiles and task-specific skills. Steward owns
persistent specialist lifecycle; ordinary usage does not require new roles.
Role SOUL boundaries are advisory, not operating-system isolation.

Fresh/incomplete installs keep dispatch off. Successful setup enables one default
gateway dispatcher with automatic review, concurrency one and no automatic
decomposition through native configuration and one gateway restart. Prove it
with the explicit `verify --dispatch-check` (a no-write researcher card the
gateway must claim without manual dispatch) or with real reviewed work.
OpenViking readiness is reported separately and does not gate core work.
Inspect configured and live dispatch separately before promising progress. If
activation is incomplete, resume `setup --team`; do not bypass it with one-shot
dispatch. Keep default responsible for the task graph and preserve queued work.
Live channel notification and actual worker artifacts still require evidence.

For independent review, inspect same-card execution history and artifacts:
implementation actor `executor`, a different approval actor `reviewer`. Done status alone
is insufficient. Request changes only for a legitimate issue, never fabricate
a rejection to satisfy a test. Do not commit or push generated work unless asked.

## Observation versus acceptance

Container health is not component readiness. Verify Hermes/team, dispatch,
development toolchain, OpenViking and configured channels independently. Coding
requires the repository's needed compiler/runtime and workdir inside `/workspace`,
not merely terminal/file tools. Configured human-facing adapters must route to
`default` with core development, Kanban and memory capabilities; users should not
need to manually enable Kanban on each channel.

RepoKit `verify` observes files, metadata, native integration configuration, and
bounded health responses. Do not add inference, dispatch, or memory writes to
make verification pass. Service health or configuration cannot establish the
following behavioral results; perform them only within an authorized live test:

| Gate | Required evidence |
| --- | --- |
| Host/default chat | Fresh host shell, unrelated cwd, native default profile and `/workspace` identity; real configured chat |
| Team | Six resolving profiles, distinct descriptions/SOULs, preserved owner memories |
| Automatic dispatch | Gateway claims a no-write researcher card and researcher completes it; no manual dispatch |
| Development runtime | Target manifest requirements match available toolchain, `/workspace` and mount access |
| Channels | Configured adapters route to default with CLI core capability parity; actual task/result delivery tested separately |
| Same-card review | Real artifact review with distinct executor/reviewer run identities |
| Shared memory | Unique durable project fact written through OpenViking, recalled in another fresh profile/session, then recalled after restart |
| Repository isolation | Normally configured second disposable repo cannot retrieve that fact; different user headers alone are insufficient |
| Runtime independence | Remove only the test-owned bootstrap binary; launcher/native commands and raw Compose restart work with profiles, board and memory preserved |

Summarize the observed capabilities, for example after the dispatch check and channel
checks passed but before private memory setup:

```text
core team           healthy
dispatch            healthy
development runtime healthy
channels            healthy
memory              pending
readiness           CORE_READY
```

`FULL_READY` adds authenticated OpenViking readiness. These summary labels do not
replace raw component evidence or certify memory recall/isolation, independent
review or removal-first acceptance. If dispatch is actually off, the dispatch check
never passed, a required compiler is missing or configured channels fail, report the core
blocker instead of inferring readiness from a healthy container.

For self-dogfood, preserve repository source. For a disposable release fixture,
removing its copied source can also be part of the explicit acceptance scope.
Never delete a system-installed RepoKit binary to manufacture removal evidence.
Check generated artifacts for dependencies on the removed binary path.

Report each requested gate as PASS, FAIL, BLOCKED, or NOT TESTED, with evidence.
Compare final repository state/content with the baseline and run `git diff
--check`. State exactly what changed, including host links and runtime files.
If private core-provider setup blocks progress, finish independent preparation
and provide the concrete private command and completion signal. Pending optional
memory setup is reported separately. Do not call partial
bootstrap success a passed full-stack dogfood.
