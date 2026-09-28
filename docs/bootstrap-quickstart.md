# Bootstrap quickstart and native handoff

The Hermes, OpenViking and local Laya deployment is implemented. `plan` inspects without writing;
`install` publishes Compose, native defaults and a standalone launcher. It prints
the exact Compose start command, including the selected Docker context.

```sh
cd my-project
hermes-repokit plan
hermes-repokit install
# Run the printed Compose build/start command for all three services.
hermes-repokit install          # native Kanban initialization
hermes-repokit setup
hermes-repokit verify
.hermes/bin/hermes-my-project
```

The default roster is default/researcher/planner/executor/reviewer/steward.
`--engineering` is a legacy alias. `setup` explicitly selects default and requires
a real interactive terminal before cloning missing specialists. Existing
profile edits are preserved and reported as drift. `default` is the normal user
entry point; it delegates team changes to steward. See the [team model](team-model.md).
Plain `setup` continues with private shared-memory setup, then pinned native
Nerve configuration and real local Laya checks for all six roles. Failed steps
preserve native state. Resume with `setup --memory` or `setup --supervision`.
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
docker compose --env-file /dev/null -f .hermes/compose.yaml up -d --force-recreate hermes laya
docker compose --env-file /dev/null -f .hermes/compose.yaml logs
docker compose --env-file /dev/null -f .hermes/compose.yaml pull hermes openviking
```

When using a named context, put `--context NAME` between `docker` and `compose`.
The launcher always uses its captured context and absolute Compose path.
A stopped service produces its native error; RepoKit does not auto-start it.
The generated foundation currently uses `sleep infinity` to keep native exec
available before credentials exist. Starting the native gateway and dispatch
is an operator action and still needs full release qualification.

`down` preserves mounted state. Do not use `down -v` as routine recovery.
Hermes/OpenViking image pulls retain their stored immutable digests. Laya builds
from `.hermes/laya-image` with pinned dependencies, upstream archive and model
hashes. Its model weights persist in the image; writable caches persist in
`.hermes/laya`, and runtime model downloads are disabled. Laya shares Hermes's
network namespace on loopback port 8765 with no published host port. Recreate
Hermes and Laya together when replacing either container:

```sh
docker compose --env-file /dev/null -f .hermes/compose.yaml up -d --force-recreate hermes laya
```

Updating pins requires new qualification. An explicitly qualified local content
image ID is also supported; see [packaging](../packaging/laya/README.md).

## OpenViking configuration

Normal installation includes the pinned official v0.4.21 sidecar and a private
`.hermes/openviking` directory mounted at `/app/.openviking`. No host port is
published. Installation never starts it or activates the Hermes memory provider.
The official entrypoint returns HTTP 503 until native configuration exists.

For an existing installation, rerun `install`. Only exact generated Hermes-only
or Hermes/OpenViking Compose is upgraded to the three-service default. Prior
Compose is saved as `.hermes/compose.hermes-only.yaml` or
`.hermes/compose.before-laya.yaml`. Edited Compose/build recipes, unknown sidecar
data and conflicting backups are preserved and refused. Native profiles and credentials
are not replaced. A previous exact backup permits resuming interrupted directory
preparation. Start all services with the context-specific command printed by `install`.
Plain `setup` completes default/team setup before entering this memory flow.
To resume memory setup alone in your own terminal:

```sh
hermes-repokit setup --memory
```

This delegates to native `openviking-server init` only when `ov.conf` is absent,
then native `openviking-server doctor`, validates the server configuration,
restarts OpenViking and waits for authenticated health before invoking
`hermes -p default memory setup openviking`. Existing server configuration is
preserved. Doctor may call your configured model services; this is deliberate
setup behavior. `verify` never performs those calls. A failed or cancelled step
leaves native state available for inspection and a later rerun.

During native server setup, select remote binding `0.0.0.0`, port `1933` and
API-key authentication. Decline the wizard's offer to start a second server;
the container entrypoint owns that process. Configure actual embedding and extraction/VLM providers
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
`http://openviking:1933`, the normal repository user key and **Mirror to
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
docker compose --env-file /dev/null -f .hermes/compose.yaml exec openviking openviking-server init
docker compose --env-file /dev/null -f .hermes/compose.yaml exec openviking openviking-server doctor
```

Native synchronization and automatic extraction are intended behavior. Successful
setup, a healthy service or a working resolver is not proof of durable recall.
Real cross-profile write/recall, restart persistence and cross-repository denial
remain the [live qualification gate](qualification/generic-team-memory.md).

## Native local supervision

After memory setup, plain `setup` checks the running local Laya service, admits
the pinned upstream Nerve plugin through native installation with `--no-enable`,
configures its local-only settings, and then enables it in all six profiles.
Fresh profile-scoped interpreters check the upstream hooks and actual
`LOCAL_ONLY` decisions. Setup requires a quiescent board with dispatch disabled,
then recreates Hermes and Laya together, waits for health, and repeats the native
checks before reporting success. A stopped Laya service fails closed; no hosted fallback
is configured. To resume this step after default/team setup:

```sh
hermes-repokit setup --supervision
```

Existing conflicting native plugin settings, source drift, external/managed
plugin selections and proxy/environment overrides are preserved and refused.
Dispatch must be disabled and running/review cards resolved before supervision
setup. Reopen manually attached native sessions after the coordinated recreation. See
[supervision setup](../internal/native/supervision.py),
[CLI coordination](../internal/cli/supervision.go) and the
[executed six-role fixture](qualification/runtime-integrations.md).

## Qualification boundaries

`verify` reads files/metadata and bounded Docker observations. It never invokes
Hermes commands that might initialize a database, migrate state, refresh auth,
dispatch work or perform inference/extraction. Owner-edited launchers are
preserved, but their Docker context is reported unknown unless their routing
can be proved. The receipt is not consulted. Memory health uses authenticated
read-only `GET /health`; Laya uses `GET /healthz`. These observations do not load
plugins or invoke models. `review` remains `unqualified`, so a healthy stack does
not produce a successful whole-deployment verification exit status.

The release must still remove a disposable RepoKit binary AND checkout, remove
the receipt, change directory, use native chat/commands, restart with raw
Compose, perform actual bounded executor→distinct-reviewer work, restart again,
and prove sessions/board/memory/Nerve/Laya persistence. The current offline
independence test uses the actual CLI, then deletes its copied source/binary and
receipt. The Docker foundation test passes real CLI install/verify/rerun and
native exec/restart persistence after removing that source/binary. Neither test
claims authenticated chat, memory recall or independent review. The separate
six-role supervision fixture does prove real local inference, offline failure
and recreation after installer removal; its scope remains narrower than this
full integrated gate.
Dogfood on RepoKit itself follows that full gate. No self-apply is required.
