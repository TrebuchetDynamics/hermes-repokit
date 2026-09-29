# Install, resume or migrate RepoKit

## Establish the target and preserve it

Resolve the user-specified path or `git rev-parse --show-toplevel`; change to
that root before every RepoKit command. Inspect repository instructions and:

```sh
git status --short --untracked-files=all
git branch --show-current
git rev-parse HEAD
git status --porcelain=v1 -z --untracked-files=all | sha256sum
```

Retain the baseline outside tracked source. For dirty files, record content
hashes as well: an unchanged status listing alone cannot prove preservation.
Do not reset, stash, clean, commit, or discard existing work for installation.

Check Docker/Compose availability and the selected local context. Inspect
existing `.hermes` ownership, tracked private-state paths, generated routing,
and matching container mounts without dumping credential files or Docker env.
An existing valid deployment is a resume, not a second installation.

## Legacy topology migration

Inventory the generated topology, version, captured Docker context and persistent
mounts before mutating an older OpenViking/Laya sidecar deployment. Preserve owner
profiles, credentials, sessions, board and memory. Use the selected RepoKit
revision's documented reconciliation path to generate replacement configuration;
there is no assumed `migrate` command.

Do not manually delete legacy containers before replacement configuration exists.
If the supported path requires quiescing a legacy memory writer before publication,
use its original Compose/native lifecycle within the authorized migration scope;
stopping a writer is not permission to delete its data or containers. Never run two
memory processes against the same database. Backups, replacement routing and
rollback/recovery must be concrete before any separately authorized destructive step.

An installer refusal is evidence that this topology is not safely reconciled by
that version. Preserve it and report the exact unsupported preimage or owner drift.
Request the bounded missing source migration or owner decision, rather than
hand-editing Compose, renaming state or deleting the old deployment to force adoption.

## Obtain the bootstrap executable

Reuse a known RepoKit executable after checking its provenance and `--help`.
Otherwise use a verified local RepoKit source checkout. If none exists, clone
`https://github.com/TrebuchetDynamics/hermes-repokit.git` into a separate temporary
source directory, honor a requested revision, and record the checked-out commit.
Do not clone RepoKit over the target or update an existing checkout implicitly.

Build from that source root using its declared Go toolchain (`go.mod`; the
checked contract requires Go 1.26+). Use a fresh temporary directory for the
binary so an unrelated `/tmp/hermes-repokit` is not overwritten:

```sh
# Run in the verified RepoKit SOURCE checkout.
set -e
repokit_build_dir=$(mktemp -d "${TMPDIR:-/tmp}/repokit-bootstrap.XXXXXX")
CGO_ENABLED=0 go build -o "$repokit_build_dir/hermes-repokit" ./cmd/hermes-repokit
"$repokit_build_dir/hermes-repokit" --help
```

The bootstrap is pure Go; ordinary installation does not need a host C compiler.
Contributor validation (`go test ./...`, `go test -race ./...`, vet and formatting)
belongs to source changes, not every installation from a verified revision.
If developing RepoKit, run the applicable checks and report any unavailable race
toolchain separately; use a suitable development environment for that check.
Do not demand a host compiler installation or declare installation blocked merely
because the race detector cannot build.

Carry that absolute binary path into subsequent commands and the private setup
handoff. Do not assume shell variables survive tool calls or exist in the
owner's terminal. Go is needed for source builds, not the generated runtime.
Missing Go/Docker or unsupported platforms are concrete prerequisites; do not
silently install host packages. The generated development image targets
Linux amd64; a different platform needs qualification, not guessed substitutions.

## Plan, publish, and start

From the **target root**, run the acquired binary's `plan`. Inspect its target,
container, project, launcher, Docker context, collisions, and unsupported items.
Refuse foreign/symlinked state, unknown generated-file edits, name collisions,
or unresolved ownership. Existing root Compose files belong to the project and
must coexist with RepoKit's `.hermes/compose.yaml` and separate project namespace.
Do not rename, merge, edit or start them as part of RepoKit installation. Use the
printed explicit file/context command; never an unqualified `docker compose up`.
If an older RepoKit revision rejects a root Compose file, use a compatible revision
within the authorized source-selection scope instead of modifying the owner's stack.

If safe, run `install`. It publishes `.hermes/compose.yaml`, native defaults,
`.hermes/bin/hermes-<repo>`, embedded OpenViking storage scaffolding and
the pinned development-image recipe. It preserves recognized prior state and refuses
ambiguous changes. It does not start services.

Fresh config trusts `/workspace` for native repository skills, and managed profile
reconciliation carries that trust across the team. Preserve explicit discovery
opt-outs, disabled skills and other trusted roots; native scanning remains active.

Run the **exact all-service Compose build/start command printed by install**,
including context, absolute file, `--env-file /dev/null`, and selector cleanup.
This may download/build the pinned development image. Inspect the same project's
`compose ps`, then rerun `install` for native Kanban initialization and existing
team reconciliation. Do not handwrite a competing Compose deployment, change
memory provider or create a RepoKit runtime daemon.

## Private setup and resumption

Give the owner one concrete command, with actual quoted absolute paths:
`cd '<target-root>' && '<bootstrap-binary>' setup`.
The owner runs it in their private terminal; request only a completion signal,
not transcripts, screenshots, tokens, or keys. Keep the binary available until
bootstrap is complete. Do not use a captured PTY to conduct private setup.

Plain `setup` is the private full-configuration entry point. Resume team reconciliation
with `setup --team`, or memory setup with `setup --memory`;
those flags are mutually exclusive. Preserve existing profiles, memories,
credentials, and owner choices on every rerun.

The intended core-only path is team reconciliation from an already configured default.
Memory configuration is a separate stage; confirm the selected version implements
the separation described below before promising that core setup will succeed.

For OpenViking, follow the selected revision's `docs/bootstrap-quickstart.md`:
native server init/doctor; real owner-selected embedding and extraction/VLM
models; persistent `/opt/data/openviking/data`; private API-key authentication.
Use the native account `repokit` and normal repository user derived from `plan`.
OpenViking runs inside the Hermes container under native s6 supervision.
Hermes connects at `http://127.0.0.1:1933` using that user's key, not a root/admin
key, and no agent/peer partition. Existing native accounts/connections must be
inspected rather than recreated. Local Hermes memory remains enabled; native
OpenViking extraction/synchronization is intended behavior. Doctor may call the
configured providers; `verify` does not.

Core setup reconciles the seven profiles, then sets the native dispatch policy and
restarts the gateway. It does not prove a worker ran: use the explicit
`verify --dispatch-check` (no-write researcher card, `metadata.first_line` must
equal the README's first line) or real reviewed work.
Only then report automatic execution/review operational. Active/finalizing workers defer setup;
queued work is preserved. Optional plugins remain owner-managed through native
admission. Respect scanner refusal and preserve existing plugin choices.

## Readiness and version gaps

OpenViking is an independent capability. Pending private setup or a memory outage
does not by itself disqualify observed core engineering readiness. Report the
failing memory component and its recovery step while keeping working team
execution available; do not change providers, disable dispatch or demand memory
credentials merely to clear a warning.

Check the selected revision's actual setup order and activation checks. A version
that blocks dispatch activation when memory fails does not implement this contract.
Report dispatch as blocked by that implementation/version and memory separately.
It needs a source-level decoupling or a compatible revision, not a manual config
bypass, one-shot dispatch or a fabricated `CORE_READY`. An operating request alone
does not authorize that source change. Avoid rerunning a coupled setup against an
otherwise working gateway solely to repair optional memory; it may suspend dispatch.

After setup, run `verify` and classify each component. A current result of
`review: unqualified` keeps its exit status nonzero even when other components
are healthy. Inspect the component output rather than using the exit code as the
entire readiness decision. `CORE_READY` requires evidenced core/team, dispatch,
toolchain and configured-channel readiness; `FULL_READY` additionally requires
authenticated memory readiness. Neither label proves full live acceptance.

Configured human-facing channels must route to `default` and have the same core
development, Kanban and memory capabilities as CLI, retaining channel-specific
extras and authorization. Use RepoKit reconciliation or native Hermes configuration;
do not hand-toggle platform tool checkboxes to manufacture parity. Refresh sessions
after reconciliation, and verify an originating-channel task/result separately.

## SELinux bind mounts

On SELinux-enabled Linux hosts RepoKit detects the state during `plan`/`install`
and generates Docker's private `Z` relabeling for the two repository-owned bind
mounts, `<repo>` → `/workspace` and `<repo>/.hermes` → `/opt/data`. `verify`
reports the host state plus in-container `/workspace` and `/opt/data` access. This
is normal installation compatibility: an authorized `install` applies it without a
separate approval prompt.

`Z` is correct for the one-container topology and relabels only those two sources.
RepoKit refuses broad sources (`/`, `/home`, `/usr`, `/etc`, the user home) rather
than relabeling them. On hosts without SELinux the option is omitted and the
generated Compose is unchanged. If a running container's repository mount is
denied, `verify` reports the access probe as degraded.

Never disable SELinux (`setenforce 0`), change global policy, install custom policy
modules, relabel unrelated host paths, or hand-edit `.hermes/compose.yaml` to work
around a denial; ask the owner before any of those. For an existing deployment
whose generated Compose lacks the relabel option, detect the drift, regenerate
through `install`, and recreate the Hermes container; preserve repository source,
`.hermes` state, credentials, profiles, Kanban and OpenViking data.

## Automatic host command

An installation request includes safe host-command exposure. Current `install`
creates `~/.local/bin/<plan-launcher-name>` as a symlink to the absolute generated
launcher after publishing it. Missing `.local`/`bin` directories are created when
safe; matching links are reused. Unrelated entries, including dangling symlinks,
are never replaced. The home and destination directories must be owned and not
group/other writable; redirected `.local`/`bin` directories are refused.

Inspect installer output separately from runtime readiness. A directory missing
from PATH still receives the link, with an explicit warning and absolute command.
Unusable host directories or destination conflicts leave the local launcher
available and produce a warning. Existing PATH collisions fail preflight. Preserve
conflicts and use the reported absolute launcher; retry through `install` after
the owner resolves the condition. Do not require separate approval for ordinary
automatic link publication within an authorized installation.

Never create Bash aliases or edit shell rc files or PATH automatically. Check
`command -v` and `type -a` if an existing alias/function shadows the command;
preserve those owner definitions. Older revisions may only generate the local
launcher: report that version gap instead of claiming a host command exists.

Test command resolution in a fresh user shell, then `--help`, `kanban list`,
and `plugins list` from another directory. Bare chat requires successful private
model setup before it can be called usable. The link alone proves no chat or
integration acceptance.

Source contract: use the selected RepoKit checkout's `README.md` and `docs/bootstrap-quickstart.md`; match the installed CLI revision.
