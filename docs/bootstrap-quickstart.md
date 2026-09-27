# Bootstrap quickstart and native handoff

The Hermes-only foundation is available. `plan` inspects without writing;
`install` publishes Compose, native defaults and a standalone launcher. It prints
the exact Compose start command, including the selected Docker context.

```sh
cd my-project
hermes-repokit plan
hermes-repokit install
# Run the Compose start command printed above.
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
Plugins and sidecars remain separate qualification work. No PATH link or shell rc edit
is automatic. The repository basename determines the full launcher/container
name; collisions refuse rather than silently adding suffixes.

## Ordinary runtime management

With interfering Compose selectors unset, use the same local Docker context
recorded in the generated launcher. From the repository:

```sh
unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES

docker compose --env-file /dev/null -f .hermes/compose.yaml up -d
docker compose --env-file /dev/null -f .hermes/compose.yaml down
docker compose --env-file /dev/null -f .hermes/compose.yaml restart
docker compose --env-file /dev/null -f .hermes/compose.yaml logs
docker compose --env-file /dev/null -f .hermes/compose.yaml pull
```

When using a named context, put `--context NAME` between `docker` and `compose`.
The launcher always uses its captured context and absolute Compose path.
A stopped service produces its native error; RepoKit does not auto-start it.
The generated foundation currently uses `sleep infinity` to keep native exec
available before credentials exist. Starting the native gateway and dispatch
is an operator action and still needs full release qualification.

`down` preserves mounted state. Do not use `down -v` as routine recovery.
Image pulls retain the stored immutable digest; updating a digest requires
new qualification, not a moving `latest` deployment.

## Later OpenViking configuration (not installed by the foundation)

Once upstream integration qualification is complete, the planned official service mounts `.hermes/openviking` at `/app/.openviking` and
publishes no host port. When provisioned, run native setup in that service:

```sh
docker compose --env-file /dev/null -f .hermes/compose.yaml exec openviking openviking-server init
docker compose --env-file /dev/null -f .hermes/compose.yaml exec openviking openviking-server doctor
.hermes/bin/hermes-my-project memory setup openviking
```

The pinned v0.4.21 wizard reports `/app/.openviking/data` as its default durable
workspace. Confirm `storage.workspace` is explicitly set to that absolute path
in its `ov.conf`. Embedding and VLM setup are required; use the native wizard
for credentials. Hermes uses `memory.provider: openviking` and endpoint
`http://openviking:1933`. Configure native authorization; namespace strings are
not access control. Real cross-repository denial is an outstanding release gate.

This is separate from Hermes provider/chat setup. Native session sync and
automatic memory extraction are accepted behavior. A 503 pending-configuration
response or missing provider is not successful memory operation.

## Qualification boundaries

`verify` reads files/metadata and bounded Docker observations. It never invokes
Hermes commands that might initialize a database, migrate state, refresh auth,
dispatch work or perform inference/extraction. Owner-edited launchers are
preserved, but their Docker context is reported unknown unless their routing
can be proved. The receipt is not consulted.

The release must still remove a disposable RepoKit binary AND checkout, remove
the receipt, change directory, use native chat/commands, restart with raw
Compose, perform actual bounded executor→distinct-reviewer work, restart again,
and prove sessions/board/memory/Nerve/Laya persistence. The current offline
independence test uses the actual CLI, then deletes its copied source/binary and
receipt. The Docker foundation test passes real CLI install/verify/rerun and
native exec/restart persistence after removing that source/binary. Neither test
claims authenticated chat, memory, inference or independent review.
Dogfood on RepoKit itself follows that full gate. No self-apply is required.
