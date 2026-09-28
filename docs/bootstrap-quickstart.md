# Bootstrap quickstart and native handoff

The single-container Hermes deployment with embedded OpenViking is implemented in source. `plan` inspects without writing;
`install` publishes Compose, native defaults and a standalone launcher. It prints
the exact Compose start command, including the selected Docker context.

```sh
cd my-project
hermes-repokit plan
hermes-repokit install
# Run the printed Compose build/start command for Hermes.
hermes-repokit install          # native Kanban initialization
hermes-repokit setup
hermes-repokit verify
.hermes/bin/hermes-my-project
```

The default roster is default/researcher/planner/executor/reviewer/steward.
`--engineering` is a legacy alias. Plain `setup` explicitly selects default and requires
a real interactive terminal for private provider setup. If native setup was already
completed through the launcher, run `hermes-repokit setup --team` to provision
the six-role team from the saved default model without repeating login. This
stage does not run a private wizard or configure memory. Existing
profile edits are preserved and reported as drift. `default` is the normal user
entry point; it delegates team changes to steward. See the [team model](team-model.md).
Plain `setup` continues with private shared-memory setup and native operational activation. Failed steps
preserve native state. Resume individual stages with `setup --team`,
`setup --memory`; these flags are mutually exclusive.
After a successful stage, RepoKit admits the bundled native maintenance plugin
through Hermes's scanner and installer. Scanner refusal or edited plugin source
stops activation. Setup suspends an already operational dispatcher through native
configuration and graceful restart before reconciliation; active workers defer
setup, while queued cards are preserved.

Fresh installation keeps `dispatch_in_gateway=false`. Operational activation
requires configured native provider/model resolution for all six profiles, current
SOULs/descriptions and channel tools, initialized Kanban, default gateway routing,
authenticated shared OpenViking. Memory is mandatory;
there is no implicit degraded-memory activation mode. Missing integrations leave
setup incomplete; finish the relevant private stage and rerun `setup --team`.

After the gates pass, the default gateway alone runs automatic execution and review:
`dispatch_in_gateway=true`, `review_dispatch=true`, `auto_decompose=false`,
`max_in_progress=1`, `orchestrator_profile=default`, and an explicit allowlist of
all six permanent profiles. Specialists keep gateway dispatch off. Setup uses
native gateway start/restart, checks the replacement process, its singleton lock
and startup concurrency, and creates a no-write researcher canary. The gateway
must claim it, run researcher and complete with the README title and no changed
files; when `README.md` is absent or has no nonempty `# ` heading, it must instead
report `NO_MARKDOWN_TITLE`. Successful canaries are archived natively. A failure preserves native
history and does not report readiness. No one-shot dispatch is used.

An unchanged operational `install` rerun rechecks activation gates and reuses the
current gateway/canary evidence; it does not restart or purchase another canary.
After a failed canary, `setup --team` can retry without another provider login.
Only an exact RepoKit canary with closed researcher runs and no dependencies can
lead to a retry, under the native claim fence. Previous terminal cards are left
untouched; retry keys follow their task IDs so an interrupted retry is reused
after gateway replacement. Recovery checks at most 32 terminal predecessors.
Owner-modified cards and active/finalizing workers stop recovery.

`verify` is read-only and reports configured dispatch, live dispatch, policy and
canary qualification separately. Config enabled with an old process or missing
lock/startup evidence is stale and fails verification. Native emergency pause is
preserved. A fresh Telegram conversation (`/new`) refreshes the coordinator SOUL
and tools; native subscriptions and notification/wake return worker outcomes to
the originating conversation. Actual Telegram delivery and independent same-card
review still need live acceptance; a successful CLI canary cannot prove them.

Live main-model work and memory recall remain unqualified. No PATH link or shell
rc edit is automatic. The repository basename determines the full launcher/container
name; collisions refuse rather than silently adding suffixes.

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

An older generated deployment containing Laya is outside the recognized automatic
upgrade set. The new installer refuses its Compose artifacts and preserves the
existing state; rerunning install does not remove its service, cache or plugins.
Exact historical managed SOUL migration remains a separate supported operation.

Migration requires the owner to coordinate a backup of native profiles, credentials,
sessions, Kanban and memory, quiesce work, and stop/tear down the old deployment
through its original Compose/native tools. A fresh core deployment can then be
prepared with an explicit state-restoration plan. Do not hand-patch generated
Compose or private state to bypass ownership checks, delete private data as a
shortcut, or automatically remove owner-installed plugins. This source cleanup
performed no live migration or teardown.

## OpenViking configuration

Normal installation embeds the pinned official v0.4.21 runtime inside
`hermes-<repo>`, supervised by the existing native s6 supervisor. The private
`.hermes/openviking` directory remains mounted at `/app/.openviking` inside
Hermes. OpenViking binds to loopback; no host port or independent Compose
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
hermes-repokit setup --memory
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
and credentials. Confirm `storage.workspace` is `/app/.openviking/data`; all
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
copy keys into six profile files.

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
Compose, perform actual bounded executor→distinct-reviewer work, restart again,
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
hermes-repokit plan --docker-tests
hermes-repokit install --docker-tests
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
