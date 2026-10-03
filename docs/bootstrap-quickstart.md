# Bootstrap quickstart and native handoff

The single-container Hermes deployment is implemented in source. `plan` inspects without writing;
`install` publishes Compose, native defaults and a standalone launcher, then
builds and starts Hermes through ordinary Compose with the selected Docker context.

Existing `compose.yaml`, `docker-compose.yml` and override files can stay in the
repository root. RepoKit uses `.hermes/compose.yaml`, its own project namespace
and explicit routing; it neither merges nor manages the application's stack.
`install` builds and starts Hermes itself with that Compose file and context;
`repokit stop` and `repokit start` stop and restart it later.
Building the pure-Go bootstrap with `CGO_ENABLED=0 go build ./cmd/hermes-repokit`
requires Go, not a host C compiler. Race tests are contributor validation.
Install the bootstrap with Go 1.26+ from the latest release:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/v0.2.10/install.sh | REPOKIT_REF=v0.2.10 sh
```

To build unreleased `main` instead, drop `REPOKIT_REF` and fetch the script from
`main`. It downloads and builds the selected source, then installs the bootstrap as
`~/.local/bin/hermes-repokit` plus the `repokit` alias. Running `./install.sh`
from a source checkout builds that checkout instead. The commands below use
`repokit`, which is always the bootstrap. Where a generated `hermes-<repo>`
launcher already owns `hermes-repokit`, the installer preserves that launcher and
reports the blocked name instead of replacing it.

```sh
cd my-project
repokit plan
repokit install          # builds and starts the container, then continues into setup:
                         # private Hermes setup, team, dispatch and canary
hermes-my-project        # or .hermes/bin/hermes-my-project
```

`install` is the whole first-time path: in a terminal it continues straight into
`setup` (`--no-setup` stops after install; without a terminal it stops and
points at `repokit setup`). Neither asks you to run Compose, rerun install or
pass extra flags. `setup` ends with `RepoKit ready.`
once the canary card has run through automatic dispatch.

The default roster is default/researcher/planner/executor/tester/reviewer/steward.
Fresh installation trusts `/workspace` for native project-local skills before
the container starts. Team provisioning carries that trust to all seven profiles;
existing managed profiles gain it on the next `install` or `setup`. Hermes still scans
project skills and honors disabled skills. Existing trusted paths, skill settings
and an explicit `skills.project_discovery: false` are preserved. Start a fresh
conversation after reconciliation to refresh the skill index.

`setup` starts the deployment if it is
stopped, then hands your interactive terminal to Hermes's own private setup for
`default` only while default has no model; once one is saved (including setup
completed natively through the launcher) the wizard is skipped. Before it
creates a new team it states the autonomy posture and asks to confirm:
RepoKit's default runs workers without approval prompts, so they change the
repository and run commands unattended from any connected chat (Hermes's
hard-deny floor still applies); answering no keeps Hermes's approval prompts.
It then provisions the seven-role team, activates dispatch and runs the canary: one
no-write researcher card the gateway must claim and complete by itself (a small
model call; `--no-canary` skips it and leaves dispatch unproven). RepoKit never
configures memory. Owner-customized
profiles (changed SOUL or description) are preserved and reported without
failing; missing managed configuration is reported as drift. `default` is the normal user
entry point; it delegates team changes to steward. See the [team model](team-model.md).
Plain `setup` activates the core team. Memory setup is user-managed and separate.
Failed steps preserve native state; rerun `setup`. For recovery,
`setup --team` reconciles the team without ever opening the private wizard.
On an operational team (dispatch already on), reconciliation never rewrites
profiles under live workers. When no card is running, missing roles are created;
while a card runs, only `default`'s own Kanban tools on saved channels are
completed. Owner-customized profiles and an owner-changed dispatch policy are
only reported. `repokit plan` shows the
per-profile decision in its `team` section, and `repokit install
--reset-profile <role>` deliberately returns one roster profile to RepoKit's
baseline after backing up its files (preview it with `plan --reset-profile`);
see the [team model](team-model.md).

Fresh installation keeps `dispatch_in_gateway=false`. After the seven profiles
reconcile without drift, setup writes the native Kanban policy on `default` with
`hermes config set`: `review_dispatch=true`, `max_in_progress=1`,
`auto_decompose=false`, `orchestrator_profile=default`, the seven-profile
`dispatch_profiles` allowlist, and `dispatch_in_gateway=true` last. It then runs
`hermes gateway restart` and waits for a new gateway PID in `gateway status`.
Setup refuses (and leaves dispatch off) while any card is running. An
already-matching policy is left untouched with no restart; an owner-changed
policy is preserved and reported, never overwritten. A stopped gateway is not
started by setup. Memory is user-managed and never gates core work.

Setup does not run a model or claim that a worker executed. Prove the loop with
the explicit, paid `hermes-repokit verify --dispatch-check`: it creates one
no-write researcher card, requires the running gateway to claim it within 150
seconds without any manual dispatch, and accepts only a completed researcher run
whose `metadata.first_line` equals the README's first physical line (computed by
RepoKit) with no changed files. A passing card is archived; a failing card is
preserved for inspection.

`verify` is observational. It uses public `hermes config get`, `gateway status`
and `kanban list/show --json` and reports `CORE_READY` first; memory is not
reported. `CORE_READY` is `healthy` only when configuration, runtime,
toolchain, dispatch policy, gateway and channel Kanban tools are healthy **and** a
recent done card shows same-card review (an implementation run requesting review,
a later tester run forwarding it, and reviewer completing the card last). Without that evidence it is `unqualified`.
`verify` exits 0 unless core is `degraded`. Passive verify does not configure or
prove memory. A fresh
Telegram conversation (`/new`) refreshes the coordinator's tools; actual
originating-channel delivery is not observed by verify.

Live main-model work remains unqualified; memory is user-managed. The repository basename
determines the full launcher/container name; collisions refuse rather than silently
adding suffixes. RepoKit adds `hermes-` only when the normalized repository name
does not already start with it: `my-project` becomes `hermes-my-project`, while
`hermes-repokit` stays `hermes-repokit`. Development images use
`repokit/<container-name>:<recipe-fingerprint>`. The stable Compose project ID
continues to isolate each repository, and the image keeps its recipe fingerprint.

## Host command

After publishing the generated launcher, `install` automatically creates:

```text
~/.local/bin/hermes-<repo> → <repo>/.hermes/bin/hermes-<repo>
```

It creates missing `.local` and `bin` directories under the current user's home
when safe, and reuses a symlink to the same launcher on reruns. Existing files,
directories and links to other targets (including dangling links) are preserved.
For a repository whose name normalizes to `repokit`, `hermes-repokit` may already
be the RepoKit bootstrap installed by `install.sh`. `install` keeps it, reports
it and proceeds; open the team with the launcher path it prints, or remove that
copy (`repokit` stays the bootstrap) and rerun `install` to create the link.
Home and destination directories must be owned by the current user and must not
be writable by group or others; `.local` and `bin` must not redirect through
symlinks. RepoKit never changes existing directory permissions.

When `~/.local/bin` is on PATH as an absolute directory, the command works from
any working directory. For a repository named `hermes-repokit`:

```sh
hermes-repokit
hermes-repokit kanban list
hermes-repokit profile list
hermes-repokit setup
```

The link forwards native Hermes arguments to the generated Docker Compose
launcher. Native `setup` configures Hermes; the bootstrap binary's `setup` also
performs RepoKit team/integration reconciliation. The runtime must be started
before use.

In the `hermes-repokit` repository, the generated host command is
`hermes-repokit`; the source-installed bootstrap is `repokit`. Use `repokit`
for `plan`, `install`, `setup` and `verify`, and `hermes-repokit` for
native Hermes commands. The installer publishes both `hermes-repokit` and
`repokit`, but this repository's generated launcher already owns
`hermes-repokit`, so the installer preserves the launcher and reports the
blocked name. RepoKit likewise preserves an unrelated existing host executable
and reports the collision.

RepoKit recognizes only its current generation and keeps no legacy support. The
exception is the previous release: `install` upgrades a v0.2.0 deployment in
place, adding the toolchain-cache volume to a Go deployment after backing up its
Compose and recipe as `compose.before-toolchain-cache-*`, upgrades a v0.2.3
deployment to hide `.hermes` inside `/workspace` (old Compose kept as
`compose.before-state-mask-*`), and rewrites any SOUL that still matches the
digest RepoKit recorded when it wrote it (`.repokit-soul`) to the current one
while no card is running. Profiles from v0.2.3, which predates records, are
upgraded by installing v0.2.4 first. A deployment from any
earlier release, like an edited or foreign one, is refused and never migrated. To start over, stop it with
`docker compose -f .hermes/compose.yaml down`, move `.hermes` aside, and run
`repokit install` again.

If PATH lacks the directory, install still creates the link and prints its
absolute path. If host exposure is unavailable, install warns and gives the
repository-local launcher instead; published artifacts remain usable. Existing
PATH command collisions still refuse installation during preflight. Fix the
reported host condition and rerun `install` to retry link creation. RepoKit
never creates shell aliases or edits `.bashrc`, `.zshrc`, or PATH automatically.

## SELinux hosts

On SELinux-enabled Linux hosts RepoKit requests Docker's private bind relabeling
(`selinux: Z`) for the two repository-owned mounts, `<repo>` and `<repo>/.hermes`.
`plan` and `install` report the detected state; `verify` reports the state and
proves access from inside the runtime. SELinux itself, global policy and
unrelated host paths are never changed.

```text
Host security:
  SELinux:           enforcing
  bind relabeling:   enabled (private Z)
  /workspace access: healthy
  /opt/data access:  healthy
```

`Z` is private to the single RepoKit container and relabels only those two
bind sources. RepoKit refuses broad sources such as `/`, `/home`, `/usr`, `/etc`
or the user's home directory rather than relabeling them. On hosts without
SELinux the option is omitted and the generated Compose is unchanged. A running
container whose repository mount is denied is reported as `access` degraded, not
healthy.

If `verify` reports relabeling pending, rerun `install` to regenerate the
repo-local mounts and recreate Hermes. Do not
run `setenforce 0`, change global SELinux policy, install custom policy modules,
or edit `.hermes/compose.yaml` by hand; none of those are required for ordinary
target-repository bind mounts.

## Ordinary runtime management

With interfering Compose selectors unset, use the same local Docker context
recorded in the generated launcher. From the repository:

```sh
unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES

docker compose --env-file /dev/null -f .hermes/compose.yaml up -d --build
docker compose --env-file /dev/null -f .hermes/compose.yaml down
docker compose --env-file /dev/null -f .hermes/compose.yaml up -d --force-recreate hermes
docker compose --env-file /dev/null -f .hermes/compose.yaml logs
docker compose --env-file /dev/null -f .hermes/compose.yaml build --pull hermes
```

When using a named context, put `--context NAME` between `docker` and `compose`.
The launcher always uses its captured context and absolute Compose path.
A stopped service produces its native error; RepoKit does not auto-start it.
The generated foundation currently uses `sleep infinity` to keep native exec
available before credentials exist. Successful native setup activates the gateway after its readiness gates pass;
actual model work and channel delivery still need full release qualification.

`down` preserves mounted state. Do not use `down -v` as routine recovery.
The generated development image pins its Hermes input.
Recreate the single Hermes service when replacing the image; updating pins requires
new qualification. User-managed memory state stays in its own mount.

## Removing a deployment

`repokit remove` deletes a RepoKit deployment completely, including the
private `.hermes` state, after you type the repository name in an interactive
terminal. It refuses a `.hermes` whose launcher or Compose file RepoKit did not
generate (edited, foreign or hand-built) and a container that does not belong to
this repository, and it removes a host command only if it is RepoKit's symlink
to this launcher. Compose `down` always runs for the generated project, so a
network left by a killed container is cleaned too. Piped input is never accepted
as confirmation. Stop the deployment with ordinary Compose instead if you only
want it offline.

If `.hermes` is already gone (deleted, or lost after an interrupted first
install), `install` refuses and names the container left behind. `remove` then
clears only what provably belongs to this repository: the container whose
Compose project and mounts match this path, that project's volumes and
networks, RepoKit's image for it and the host link to this launcher. It asks for
the same typed confirmation; after that, `install` starts over.

## Qualification boundaries

`verify` reads files/metadata and bounded Docker observations. It never invokes
Hermes commands that might initialize a database, migrate state, refresh auth,
dispatch work or perform inference/extraction. Owner-edited launchers are
preserved, but their Docker context is reported unknown unless their routing
can be proved. The receipt is not consulted. Memory is user-managed and is not
probed by `verify`. These observations do not load
plugins or invoke models. `review` remains `unqualified`, so a healthy stack does
not produce a successful whole-deployment verification exit status.

The release must still remove a disposable RepoKit binary AND checkout, remove
the receipt, change directory, use native chat/commands, restart with raw
Compose, perform actual bounded executor→tester→reviewer work, restart again,
and prove session and board persistence. The current offline
independence test uses the actual CLI, then deletes its copied source/binary and
receipt. The Docker foundation test passes real CLI install/verify/rerun and
native exec/restart persistence after removing that source/binary. Neither test
claims authenticated chat or independent review. Dogfood on RepoKit itself follows that full gate. No self-apply is required.

## Repository development and optional Docker tests

Normal `install` now publishes `.hermes/development-image` and builds Hermes with
its project toolchain. A `go.mod` at the root or in a nested project up to three
folders deep (a monorepo's `rig-vigia/go.mod`) selects pinned Go plus
checksum-pinned `staticcheck`; a `Cargo.toml` likewise selects the official
Rust release (rustc and cargo with clippy, rustfmt and rust-analyzer), pinned by
checksum, with Cargo's registry on the toolchain-cache volume. A
`pubspec.yaml` selects the official stable Flutter SDK with its Dart, pinned by
the checksum in Flutter's release manifest; the build warms `flutter analyze`
and `flutter test` on a scratch project, and pub's cache lives on the same
volume. Flutter publishes Linux SDKs for x86_64 only, so on arm64 it is reported
missing. An app with a `linux/` runner also gets the Linux desktop toolchain
(clang, ninja, GTK 3 headers and Xvfb for headless widget tests) from Debian
packages pinned to a fixed snapshot.debian.org date, warmed by a scratch
`flutter build linux`. A Godot 4 `project.godot` selects the Godot minor its
`config/features` declares (4.4 to 4.7): every official stable patch of that
minor is installed under its upstream name (`Godot_v4.5.1-stable_linux.x86_64`),
each verified against Godot's published SHA-512, with `godot` naming the newest,
so a repository that pins an exact patch finds it on PATH. The build runs a
headless import and script. A second project on another minor, an unqualified
minor or a Godot 3 project is reported by `verify`. Export templates are not
provisioned. Android, iOS, macOS and Windows builds are not provisioned. A
`rust-toolchain` file pinning another release, or a newer `rust-version`, is
reported by `verify`. Vendored, generated and hidden trees are skipped and
symlinks are never followed. Node/npm, Python and standard build utilities come
from the pinned base plus checksum-pinned tools. JVM projects (such as an app's
Android wrapper) are noted but not provisioned.
`install` builds and starts it. A changed recipe (such as a repository gaining
a `go.mod`, `Cargo.toml` or `pubspec.yaml`) is republished in place with a retained backup; edited recipes are
preserved and refused.
`verify` reports actual development tool versions and profile workdirs separately
from memory and model-driven acceptance.

Inside the container, the repository's own `.hermes` is hidden under
`/workspace` by an empty read-only mount; private state is reached only through
`/opt/data`. Agents' whole-tree commands in the repository (`grep -r`, linters)
never walk sessions, logs or credentials, and Hermes never mistakes default's
skills folder for repository skills. Skills every profile should share belong in
the repository's own `.agents/skills/`.

Go's module and build caches live on a project-scoped `toolchain-cache` volume
mounted at `/var/cache/repokit`, not in the repository, so whole-tree commands
such as `gofmt -l .` or `grep -r` never walk third-party sources. The volume
survives restarts and recreation and is removed with the deployment. Earlier
releases kept these caches in `.hermes/development`; `install` reports that
directory once it is unused, with a removal command, and never deletes it.

For a repository that needs Docker integration tests, explicitly select:

```sh
repokit plan --docker-tests
repokit install --docker-tests   # builds and (re)creates the development image
# Run the printed --profile docker-tests daemon start command (privileged, opt-in).
```

This adds a privileged test daemon with its own project-scoped volumes and no
host Docker socket. It is not a strong host-kernel security boundary. No daemon
is started by install. Once activated, authorized Hermes tasks can invoke
`repokit-docker-test` from `/workspace`; it runs tests in a disposable clean
source snapshot, with matching client/daemon scratch paths. It needs neither
RepoKit nor host Go at runtime. See [development qualification](qualification/development-runtime.md)
for the supported matrix and the still-pending live acceptance gates.
