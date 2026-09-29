# Bootstrap quickstart and native handoff

The single-container Hermes deployment is implemented in source. `plan` inspects without writing;
`install` publishes Compose, native defaults and a standalone launcher. It prints
the exact Compose start command, including the selected Docker context.

Existing `compose.yaml`, `docker-compose.yml` and override files can stay in the
repository root. RepoKit uses `.hermes/compose.yaml`, its own project namespace
and explicit routing; it neither merges nor manages the application's stack.
Use the printed start command so the intended Compose file and context are selected.
Building the pure-Go bootstrap with `CGO_ENABLED=0 go build ./cmd/hermes-repokit`
requires Go, not a host C compiler. Race tests are contributor validation.
Install the bootstrap with Go 1.26+ from the published script:

```sh
curl -fsSL https://raw.githubusercontent.com/TrebuchetDynamics/hermes-repokit/main/install.sh | sh
```

It downloads and builds the current source, then installs the bootstrap as
`~/.local/bin/hermes-repokit` plus the `repokit` alias. Running `./install.sh`
from a source checkout builds that checkout instead. The commands below use
`repokit`, which is always the bootstrap. Where a generated `hermes-<repo>`
launcher already owns `hermes-repokit`, the installer preserves that launcher and
reports the blocked name instead of replacing it.

```sh
cd my-project
repokit plan
repokit install
# Run the printed Compose build/start command for Hermes.
repokit install          # native Kanban initialization
repokit setup
repokit verify
.hermes/bin/hermes-my-project
```

The default roster is default/researcher/planner/executor/tester/reviewer/steward.
Fresh installation trusts `/workspace` for native project-local skills before
the container starts. Team provisioning carries that trust to all seven profiles;
existing managed profiles gain it during `setup --team`. Hermes still scans
project skills and honors disabled skills. Existing trusted paths, skill settings
and an explicit `skills.project_discovery: false` are preserved. Start a fresh
conversation after reconciliation to refresh the skill index.

`--engineering` is a legacy alias. Plain `setup` explicitly selects default and requires
a real interactive terminal for private provider setup. If native setup was already
completed through the launcher, run `repokit setup --team` to provision
the seven-role team from the saved default model without repeating login. This
stage does not run a private wizard or configure memory. Owner-customized
profiles (changed SOUL or description) are preserved and reported without
failing; missing managed configuration is reported as drift. `default` is the normal user
entry point; it delegates team changes to steward. See the [team model](team-model.md).
Plain `setup` activates the core team. Memory setup is user-managed and separate.
Failed steps preserve native state. Resume team provisioning with `setup --team`.
On an operational team (dispatch already on), reconciliation never rewrites
profiles under live workers. When no card is running, untouched earlier RepoKit
SOULs upgrade and missing roles are created; while a card runs those upgrades
are deferred and reported, and only `default`'s own Kanban tools on saved
channels are completed. An unmodified six-profile release is upgraded the same
way when idle: `setup --team` creates `tester`, upgrades the managed SOULs and
then widens the allowlist with one gateway restart. Owner-customized profiles
and an owner-changed dispatch policy are only reported.

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
and `kanban list/show --json` and reports `CORE_READY` first, marking memory as
user-managed. `CORE_READY` is `healthy` only when configuration, runtime,
toolchain, dispatch policy, gateway and channel tools are healthy **and** a
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
for `plan`, `install`, `setup --team` and `verify`, and `hermes-repokit` for
native Hermes commands. The installer publishes both `hermes-repokit` and
`repokit`, but this repository's generated launcher already owns
`hermes-repokit`, so the installer preserves the launcher and reports the
blocked name. RepoKit likewise preserves an unrelated existing host executable
and reports the collision.

For an exact earlier generated deployment, `install` saves the old Compose as
`compose.before-names.yaml`, publishes the new launcher and image name, and leaves
the old launcher and any existing host link usable. Native profiles, sessions,
Kanban and memory data stay in the same directories. Interrupted publication can
be retried; edited launchers, recipes, backups and conflicting names are preserved
and refused. Run the printed Compose build/start command to recreate the existing
service under its new name, then rerun the bootstrap's `install`. Do not manually
rename the live container or edit generated Compose. Older recipe upgrades retain
their version-specific backup names.

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
repo-local mounts and recreate Hermes with the printed Compose command. Do not
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

`hermes-repokit remove` deletes a RepoKit deployment completely, including the
private `.hermes` state, after you type the repository name in an interactive
terminal. It refuses a `.hermes` whose launcher or Compose file RepoKit did not
generate (edited, foreign or hand-built) and a container that does not belong to
this repository, and it removes a host command only if it is RepoKit's symlink
to this launcher. Compose `down` always runs for the generated project, so a
network left by a killed container is cleaned too. Piped input is never accepted
as confirmation. Stop the deployment with ordinary Compose instead if you only
want it offline.

## Legacy deployment migration

`plan` recognizes the exact historical Hermes/Laya local-build stack
generated by `ce7b6c8`. After a native backup and quiescing work, stop its legacy
services through the original Compose lifecycle. `install` then preserves
the original Compose as `compose.before-core.yaml` and generates the development
runtime. Build/recreate Hermes with the printed command, then run `setup --team`.
Profiles, credentials, board history and model caches remain in place.
The installer never stops/deletes legacy containers or removes owner-installed plugins.

Edited or other unrecognized legacy Compose still refuses automatic adoption.
Preserve it and establish a supported migration path. Do not hand-patch generated
Compose or private state to bypass checks, or delete data to force an installation.

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
and prove sessions/board/memory persistence. The current offline
independence test uses the actual CLI, then deletes its copied source/binary and
receipt. The Docker foundation test passes real CLI install/verify/rerun and
native exec/restart persistence after removing that source/binary. Neither test
claims authenticated chat, memory recall or independent review. Dogfood on RepoKit itself follows that full gate. No self-apply is required.

## Repository development and optional Docker tests

Normal `install` now publishes `.hermes/development-image` and builds Hermes with
its project toolchain. A root `go.mod` selects pinned Go; Node/npm, Python and
standard build utilities come from the pinned base plus checksum-pinned tools.
Use the printed `--build` start command. Existing exact generated Compose may be
upgraded with a retained backup; edited recipes are preserved and refused.
`verify` reports actual development tool versions and profile workdirs separately
from memory and model-driven acceptance.

For a repository that needs Docker integration tests, explicitly select:

```sh
repokit plan --docker-tests
repokit install --docker-tests
# Run the printed development-image build/recreation command.
# Run the printed --profile docker-tests daemon start command.
```

This adds a privileged test daemon with its own project-scoped volumes and no
host Docker socket. It is not a strong host-kernel security boundary. No daemon
is started by install. Once activated, authorized Hermes tasks can invoke
`repokit-docker-test` from `/workspace`; it runs tests in a disposable clean
source snapshot, with matching client/daemon scratch paths. It needs neither
RepoKit nor host Go at runtime. See [development qualification](qualification/development-runtime.md)
for the supported matrix and the still-pending live acceptance gates.
