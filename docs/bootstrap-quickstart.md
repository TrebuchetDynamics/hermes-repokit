# Bootstrap quickstart and native handoff

The single-container Hermes deployment with embedded OpenViking is implemented in source. `plan` inspects without writing;
`install` publishes Compose, native defaults and a standalone launcher. It prints
the exact Compose start command, including the selected Docker context.

Existing `compose.yaml`, `docker-compose.yml` and override files can stay in the
repository root. RepoKit uses `.hermes/compose.yaml`, its own project namespace
and explicit routing; it neither merges nor manages the application's stack.
Use the printed start command so the intended Compose file and context are selected.
Building the pure-Go bootstrap with `CGO_ENABLED=0 go build ./cmd/hermes-repokit`
requires Go, not a host C compiler. Race tests are contributor validation.
From a RepoKit source checkout, `./install.sh` builds it with Go 1.26+ and
installs the bootstrap as `~/.local/bin/repokit`; it does not download a release.
The commands below use that bootstrap name.

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
stage does not run a private wizard or configure memory. Existing
profile edits are preserved and reported as drift. `default` is the normal user
entry point; it delegates team changes to steward. See the [team model](team-model.md).
Plain `setup` activates the core team, then continues with optional private shared-memory setup. Failed steps
preserve native state. Resume individual stages with `setup --team`,
`setup --memory`; these flags are mutually exclusive.
Team reconciliation on an operational team (dispatch already on) observes and
reports drift; it never rewrites profiles under live workers. It only completes
`default`'s own Kanban/memory tools on saved channels. The one exception is an
unmodified six-profile release: when no card is running, `setup --team` creates
`tester`, upgrades the six managed SOULs and then widens the allowlist with one
gateway restart. Owner-edited profiles or policy are still only reported. `setup --memory` never
touches dispatch; a failed memory stage keeps its nonzero result without taking
core work offline.

Fresh installation keeps `dispatch_in_gateway=false`. After the seven profiles
reconcile without drift, setup writes the native Kanban policy on `default` with
`hermes config set`: `review_dispatch=true`, `max_in_progress=1`,
`auto_decompose=false`, `orchestrator_profile=default`, the seven-profile
`dispatch_profiles` allowlist, and `dispatch_in_gateway=true` last. It then runs
`hermes gateway restart` and waits for a new gateway PID in `gateway status`.
Setup refuses (and leaves dispatch off) while any card is running. An
already-matching policy is left untouched with no restart; an owner-changed
policy is preserved and reported, never overwritten. A stopped gateway is not
started by setup. OpenViking is reported independently and never gates core work.

Setup does not run a model or claim that a worker executed. Prove the loop with
the explicit, paid `hermes-repokit verify --dispatch-check`: it creates one
no-write researcher card, requires the running gateway to claim it within 150
seconds without any manual dispatch, and accepts only a completed researcher run
whose `metadata.first_line` equals the README's first physical line (computed by
RepoKit) with no changed files. A passing card is archived; a failing card is
preserved for inspection.

`verify` is observational. It uses public `hermes config get`, `gateway status`
and `kanban list/show --json` and reports `CORE_READY`, `MEMORY_READY` and
`FULL_READY` first. `CORE_READY` is `healthy` only when configuration, runtime,
toolchain, dispatch policy, gateway and channel tools are healthy **and** a
recent done card shows same-card review (an implementation run requesting review,
a later tester run forwarding it, and reviewer completing the card last). Without that evidence it is `unqualified`.
`verify` exits 0 unless core is `degraded`. Passive verify never proves memory,
so `MEMORY_READY` and `FULL_READY` are never `healthy` from it. A fresh
Telegram conversation (`/new`) refreshes the coordinator's tools; actual
originating-channel delivery is not observed by verify.

Live main-model work and memory recall remain unqualified. The repository basename
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
native Hermes commands. An independently built bootstrap named
`hermes-repokit` still needs a separate absolute path. RepoKit preserves an
unrelated existing host executable and reports the collision.

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
The generated development image pins its Hermes and embedded OpenViking inputs.
Recreate the single Hermes service when replacing the image; updating pins requires
new qualification. Private memory configuration and data stay in their mount.

## Legacy deployment migration

`plan` recognizes the exact historical Hermes/OpenViking/Laya local-build stack
generated by `ce7b6c8`. After a native backup and quiescing work, stop its legacy
memory writer through the original Compose lifecycle. `install` then preserves
the original Compose as `compose.before-core.yaml` and generates the development
runtime. Build/recreate Hermes with the printed command, then run `setup --team`.
Profiles, credentials, board history, memory and model caches remain in place.
The installer never stops/deletes legacy containers or removes owner-installed plugins.

Edited or other unrecognized legacy Compose still refuses automatic adoption.
Preserve it and establish a supported migration path. Do not hand-patch generated
Compose or private state to bypass checks, or delete data to force an installation.

## OpenViking configuration

Normal installation embeds the pinned official v0.4.21 runtime inside
`hermes-<repo>`, supervised by the existing native s6 supervisor. The private
`.hermes/openviking` directory is the durable state root at
`/opt/data/openviking` inside Hermes. OpenViking binds to loopback; no host port or independent Compose
service is created. Installation never starts services or activates the Hermes memory provider.
The official entrypoint returns HTTP 503 until native configuration exists.

Before upgrading an old sidecar installation, stop its OpenViking service
through its original generated Compose file. `install` refuses a running old
sidecar so two processes cannot write the same memory database. Keep all private
state. Exact recognized legacy Compose files can be upgraded with a backup;
owner-edited files and earlier development recipes are refused, never replaced
silently. Existing private connections using the old `http://openviking:1933`
endpoint require native `setup --memory` relinking to loopback. Existing server
configuration is preserved; the installed wrapper forces the runtime listener
to loopback. See [embedded-memory qualification](qualification/embedded-openviking.md).
Plain `setup` completes default/team setup before entering this memory flow.
To resume memory setup alone in your own terminal:

```sh
repokit setup --memory
```

This delegates to native `openviking-server init` only when `ov.conf` is absent,
then native `openviking-server doctor`, validates the server configuration,
restarts only the embedded OpenViking s6 service and waits for `/health` before invoking
`hermes -p default memory setup openviking`. Existing server configuration is
preserved. Doctor may call your configured model services; this is deliberate
setup behavior. `verify` never performs those calls. A failed or cancelled step
leaves native state available for inspection and a later rerun.

During native server setup, select **Remote** mode, port `1933` and
API-key authentication. The native wizard only creates a root key in Remote
mode; the installed service wrapper overrides its configured binding to
`127.0.0.1` for both pending and configured servers. Decline the wizard's offer to start a second server;
the native s6 supervisor owns that process inside Hermes. Configure actual embedding and extraction/VLM providers
and credentials. Confirm `storage.workspace` is `/opt/data/openviking/data`; all
service data must stay in that persistent mount. No model or budget is selected
by RepoKit. Keep automatic extraction enabled. Local storage does not imply
that your chosen models run locally.

Before completing the Hermes wizard, use OpenViking's native admin API inside
the service network to create account `repokit` and a normal user whose ID is
the repository identity printed by `plan` and `setup --memory`. The selected
server exposes `POST /api/v1/admin/accounts` with `account_id` and
`admin_user_id`, followed by `POST /api/v1/admin/accounts/repokit/users` with
`user_id` and `role: user`. These are native admin operations using the private
server root/admin key; their returned keys stay in your terminal/native state.
Do not give the root or account-admin key to Hermes. Existing accounts/users
must be inspected rather than recreated or silently rotated. See the pinned
[upstream admin API](https://github.com/volcengine/OpenViking/blob/3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1/openviking/server/routers/admin.py).

In the Hermes wizard, choose **Custom URL**, endpoint
`http://127.0.0.1:1933`, the normal repository user key and **Mirror to
OpenViking store**. Use no agent/peer. On reruns select the existing shared
connection instead of making another one. The private native connection file
lives below `/opt/data/.openviking` and remains authoritative after RepoKit is
removed. RepoKit links its path through native config commands; it does not
copy keys into seven profile files.

Before updating specialists, RepoKit checks all six effective native secret
scopes and the server-derived account/user/role. Owner-selected providers or
conflicting connection paths cause refusal. Built-in local memory remains
enabled. Partial native config writes are preserved and can be inspected with
native commands; RepoKit does not roll back credential state.

To repair server configuration manually, use the same captured Docker context:

```sh
docker compose --env-file /dev/null -f .hermes/compose.yaml exec --user hermes hermes repokit-openviking server init
docker compose --env-file /dev/null -f .hermes/compose.yaml exec --user hermes hermes repokit-openviking server doctor
```

Native synchronization and automatic extraction are intended behavior. Successful
setup, a healthy service or a working resolver is not proof of durable recall.
Real cross-profile write/recall, restart persistence and cross-repository denial
remain the [live qualification gate](qualification/generic-team-memory.md).

## Qualification boundaries

`verify` reads files/metadata and bounded Docker observations. It never invokes
Hermes commands that might initialize a database, migrate state, refresh auth,
dispatch work or perform inference/extraction. Owner-edited launchers are
preserved, but their Docker context is reported unknown unless their routing
can be proved. The receipt is not consulted. Memory health uses authenticated
read-only `GET /health`. These observations do not load
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
