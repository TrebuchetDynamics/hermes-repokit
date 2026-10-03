# Install or resume RepoKit

Work through these steps in order and stop at the first refusal; a refusal is
evidence to report, not an obstacle to route around.

## 1. Establish the target and its baseline

Resolve the user-specified path or `git rev-parse --show-toplevel`, and change
to that root before every RepoKit command. Read the repository's own agent
instructions, then record:

```sh
git branch --show-current
git rev-parse HEAD
git status --short --untracked-files=all
git status --porcelain=v1 -z --untracked-files=all | sha256sum
```

Keep the baseline outside tracked source. For dirty files also record content
hashes: an unchanged status listing alone cannot prove preservation.

Check `docker compose version` and the active Docker context. Inspect existing
`.hermes/` ownership, generated routing and matching container mounts without
dumping credential files or container environment. An existing valid deployment
is a **resume**, not a second installation.

## 2. Obtain the bootstrap CLI

Reuse an installed `repokit` if `command -v repokit` resolves and `repokit --help`
lists `plan|install|setup|verify`. Otherwise install it (Linux, Go 1.26+, curl):

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.2.16/install.sh | REPOKIT_REF=v0.2.16 sh
```

Install the latest release shown above unless the owner asks for unreleased
`main` (the same command with `main` in the URL and no `REPOKIT_REF`). The
script downloads and builds that source with `CGO_ENABLED=0`
(no C compiler needed) and publishes `~/.local/bin/hermes-repokit` plus the
`repokit` alias. It updates a bootstrap it installed earlier and preserves any
other existing command, including a generated `hermes-repokit` launcher. Read its
output: it warns when `~/.local/bin` is missing from PATH or another command
shadows the name, and prints the absolute command to use instead.

To pin a revision, or when the owner already has a RepoKit source checkout, run
`./install.sh` from that verified checkout (it builds that tree) and record its
commit. Never clone RepoKit over the target or update an owner checkout
implicitly. Contributor checks (`go test ./...`, race, vet) belong to RepoKit
source changes, not to installation.

Missing Go, Docker or an unsupported platform are prerequisites to report; do not
install host packages silently. The development image targets Linux amd64.

In later commands and handoffs, use the absolute path if `repokit` is not on
PATH; shell variables do not survive between tool calls or into the owner's terminal.

## 3. Plan

```sh
repokit plan        # JSON report; writes nothing
```

Check the target identity, container/launcher name, Docker context, detected
development requirements, SELinux state, `existing_state`, `collisions` and the
`team` section (the per-profile decision: create, upgrade, preserve customized).
Refuse on foreign or symlinked state, unknown edits to generated files, name
collisions or unresolved ownership. The owner's root Compose files stay as they
are — never rename, merge, edit or start them.

Container and launcher name: `hermes-` plus the normalized repository name,
unless it already starts with `hermes-` (`my-project` → `hermes-my-project`,
`hermes-repokit` stays `hermes-repokit`). Collisions refuse; there are no
automatic suffixes.

## 4. Install and start

```sh
repokit install
```

This publishes `.hermes/compose.yaml`, native defaults, `.hermes/bin/hermes-<repo>`,
the pinned development-image recipe and the `~/.local/bin/hermes-<repo>` link, then
builds and starts the `hermes` service through ordinary Compose (the first build
can take several minutes), waits for Hermes to answer and initializes native
Kanban. It preserves recognized prior state and refuses ambiguous changes. On an
existing deployment it recreates the container only when the image changed, and
defers that while a Kanban card is running. Dispatch stays off until setup.

Afterwards, `repokit stop` stops the deployment with all state kept and
`repokit start` brings it back (Hermes restarts a gateway that was running).
`repokit remove` deletes the deployment and its `.hermes` state after the owner
types the repository name; never run it on the owner's behalf without that intent.
If `.hermes` is already gone (deleted, or lost to an interrupted first install),
`install` refuses and names the leftover container; `remove` then clears only what
provably belongs to this path (matching container, project volumes and networks,
its image and host link) behind the same typed confirmation, and `install` starts
over.

Detected toolchains: Go (`go.mod` at the root or up to three folders deep, plus
`staticcheck`), Rust (`Cargo.toml`), Flutter/Dart (`pubspec.yaml`; Linux desktop
toolchain when a `linux/` runner exists; x86_64 only), Godot 4 (`project.godot`;
every patch of its declared minor as `Godot_v<version>-stable_linux.<arch>`,
`godot` the newest; an `export_presets.cfg` adds its platforms' export templates
under `$GODOT_EXPORT_TEMPLATES`, and an Android preset adds JDK 17 and the
Android SDK, x86_64 only), with Node/npm and Python from the base image. JVM, Android, iOS, macOS and Windows builds are not
provisioned — report them as a gap rather than installing toolchains by hand
inside the container. A repository that gains a manifest gets a republished
recipe on the next `install`.

Do not hand-write a competing Compose deployment or run an unqualified
`docker compose up`.

Optional: for a repository whose tests need Docker, use `plan --docker-tests` /
`install --docker-tests` (it rebuilds the image) and run the printed
`--profile docker-tests` daemon command. This adds a privileged, project-scoped test
daemon without the host Docker socket; tasks call `repokit-docker-test` from
`/workspace`.

## 5. Private setup (owner's terminal)

Hand the owner one concrete command with real, quoted absolute paths:

```sh
cd '/abs/path/to/target' && repokit setup
```

Ask only for a completion signal (`RepoKit ready.`) — never transcripts,
screenshots, tokens or keys. Tell the owner up front that before creating a new
team `setup` asks them to confirm the autonomy posture: workers run without
approval prompts (Hermes's hard-deny floor still applies); answering no keeps
Hermes's approval prompts. That answer is theirs. `setup` is the last step of the first-time path. It
starts a stopped deployment, hands the terminal to Hermes's own private setup
while default has no model (skipped once one is saved, including setup done
natively through `hermes-<repo> setup`), reconciles the seven profiles, writes
the native dispatch policy (`review_dispatch=true`, `max_in_progress=1`,
`auto_decompose=false`, the seven-profile allowlist, `dispatch_in_gateway=true`
last), restarts or starts the gateway, and runs the canary: one no-write
researcher card the gateway must complete by itself (a small model call).
A failed canary exits nonzero without claiming readiness.

Every rerun preserves profiles, credentials, sessions and owner choices;
owner-customized profiles are preserved and reported, never overwritten.
`setup --team` is the recovery form that never opens the wizard, and
`--no-canary` skips the paid proof.

Setup never writes profiles while a card is running and leaves an owner-changed
dispatch policy untouched.

## 6. Verify

```sh
repokit verify                    # observational
repokit verify --dispatch-check   # explicit and paid: one no-write researcher card
```

Classify every component from the JSON, not just the exit code (see
[readiness report](usage.md#readiness-report)). The dispatch check requires the
running gateway to claim the card within 150 seconds without manual dispatch and
a completed researcher run whose `metadata.first_line` equals the README's first
line; a failing card is preserved for inspection. Run it only when the owner
accepts the model cost.

Then check the host command in a fresh shell from another directory:
`command -v hermes-<repo>`, `hermes-<repo> kanban list`, `hermes-<repo> profile list`.
Bare chat is usable only after private setup succeeds.

## Host command

`install` links `~/.local/bin/hermes-<repo>` → `<repo>/.hermes/bin/hermes-<repo>`.
It creates missing `.local`/`bin` directories when safe, reuses matching links and
never replaces unrelated entries (including dangling symlinks). Home and
destination directories must be owner-owned and not group/other writable;
redirected `.local`/`bin` are refused.

A PATH gap still gets the link, plus a warning and the absolute command. Existing
PATH collisions fail preflight; preserve them and retry `install` after the owner
resolves them. Never create aliases or edit shell rc files or PATH. If an alias or
function shadows the command (`type -a hermes-<repo>`), preserve it and use the
absolute launcher.

## SELinux bind mounts

On SELinux-enabled hosts `plan`/`install` detect the state and add Docker's
private `Z` relabeling to exactly two mounts, `<repo>` → `/workspace` and
`<repo>/.hermes` → `/opt/data`. Broad sources (`/`, `/home`, `/usr`, `/etc`, the
user home) are refused rather than relabeled. `verify` reports the host state and
in-container access.

If an older deployment lacks relabeling, rerun `install`; it recreates the Hermes
service with the relabeled mounts and preserves all state. Never run
`setenforce 0`, change global policy, add policy modules, relabel unrelated paths
or hand-edit the Compose file.

## Earlier releases

RepoKit keeps no legacy support: it recognizes only its current generation,
plus v0.2.0 deployments, which `repokit install` upgrades in place (just run
it). `install` refuses a deployment from any earlier release, like an edited or
foreign one, and prints the steps to start over. Relay them to the owner and
let the owner run them: stop that deployment with
`docker compose -f .hermes/compose.yaml down`, move `.hermes` aside, then
`repokit install`. Do not hand-edit Compose, rename state, delete the old
deployment or invent a `migrate` command.
