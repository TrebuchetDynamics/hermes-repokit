# Phase 7 native defaults and engineering profiles

This is a historical qualification record for the earlier engineering scaffold.
The [generic-team qualification](generic-team.md) supersedes its roster,
credential non-cloning, and staged profile publication behavior. Current setup
uses native configuration clones after default setup and preserves interrupted
profiles under their final native names.

Qualification uses Hermes v0.21.5, OCI revision
`749220ef0007f8d87bd1531f1c24b0fe93816385`, in the immutable image recorded in
[runtime observations](runtime-observations.md). No credentials or inference
are needed for these operations. This qualifies the native-state part of Phase
7; Superpowers admission/loading remains pending an exact scanner decision.

## Native operations and evidence

The pinned image's `hermes_cli/profiles.py` creates fresh profiles with an empty
credential file unless a clone option is explicitly supplied. Its
`hermes_cli/config.py` parses structured config values as real lists/booleans.
The tested operations are:

```sh
hermes kanban init
hermes profile create researcher --no-alias --description 'Research repository questions and provide source-backed findings.'
hermes -p researcher config set kanban.dispatch_in_gateway false
hermes -p researcher config set kanban.auto_decompose false
hermes -p researcher config set toolsets '["web","file","kanban"]'
hermes -p researcher config set terminal.cwd /workspace
```

Sources at the selected revision:
[profile creation](https://github.com/NousResearch/hermes-agent/blob/749220ef0007f8d87bd1531f1c24b0fe93816385/hermes_cli/profiles.py),
[native config](https://github.com/NousResearch/hermes-agent/blob/749220ef0007f8d87bd1531f1c24b0fe93816385/hermes_cli/config.py),
[toolsets](https://github.com/NousResearch/hermes-agent/blob/749220ef0007f8d87bd1531f1c24b0fe93816385/toolsets.py).
These facts were checked against source copied from the pinned local image and
then exercised against that image; current-main documentation alone did not
establish compatibility.

| Profile | Selected toolsets | Purpose |
| --- | --- | --- |
| default | hermes-cli, kanban | Native interactive work and board tools |
| researcher | web, file, kanban | Source-backed research |
| planner | file, kanban | Scoped steps and acceptance criteria |
| builder | terminal, file, kanban | Implementation and validation |
| reviewer | terminal, file, kanban | Independent candidate review |

Tool selection expresses role intent, not a security sandbox or proof of
independent review. Native Kanban worker tools still obey Hermes's own task
context restrictions. Existing profile settings are preserved.

## Publication and interruption behavior

The first `install` publishes Compose/config/launcher and reports native
initialization pending. The operator starts ordinary Compose. A subsequent
`install` verifies the running container's immutable image, project and mounts
before invoking native initialization in that same container. There is no
second Hermes runtime or implicit service start.

The native initializer acquires `/workspace/.hermes-repokit.lock` inside the
container, and checks workspace/state/lock inode identities before writing.
This is necessary because daemon-owned Docker exec work can outlive its client;
inheriting a host lock into that client alone does not cover the native writer.
`verify` never invokes this initializer.

For a new profile, native `profile create` runs in a private temporary Hermes
home beneath `.hermes`; native `config set` applies safe booleans, toolsets and
cwd there. Only then is the profile moved without replacement into standard
`profiles/<name>`. Existing profile directories and configuration remain owner
state. Failed staging is cleaned up; a failure after a prior successful native
operation preserves that operation and reports incomplete initialization.
Existing Kanban is never reset or migrated by a rerun.

The [implementation](../../internal/native/bootstrap.go) sends the
[one-shot script](../../internal/native/bootstrap.sh) as input to the selected
container. It is not persisted as a runtime hook. No runtime imports, mounts or
callbacks depend on RepoKit.

## Executed qualification

The Docker acceptance fixture exercised real CLI default initialization,
engineering initialization, native descriptions, actual persisted boolean/list
values, absence of a root `.env` sentinel in new profiles, and byte-preservation
of owner-edited profile config on rerun. The default install created no named
engineering profiles. Source and binary removal followed by ordinary Compose
restart retained native state. Offline tests also exercise interrupted config
staging without publishing an incomplete new profile.

Native helper execution uses `docker compose exec --user hermes --env
HOME=/opt/data`; the user and home were checked in the pinned image. Invoking
its `docker/main-wrapper.sh` as root through bare Docker exec failed because
`s6-setuidgid` was absent from that exec PATH. RepoKit does not modify the image
or add a replacement exec shim to work around that observation.

## Superpowers decision still pending

Candidate SHA: `8ca22dba9a94f28898bbce59f2537ff4d87c747d`.
[Exact scanner report](superpowers-8ca22dba-scan.txt) SHA-256:
`9d949b102ff519144d708e304576ed5425492c716e5ead8bfda8d4f38a9cec78`.
The recorded result is CAUTION with 229 findings.

The pinned `hermes_cli/plugins_cmd.py` supports native interactive CAUTION
confirmation via `scan_decision_cb`; the prior non-TTY attempt declined that
confirmation. `--force`, scanner disabling, and changing source trust are not
needed or authorized. A new scan must match the approved revision and findings
before any affirmative confirmation. Actual installation, enablement, and a
fresh native-session load test remain unexecuted pending approval. Approval to
qualify a disposable container does not grant blanket approval for all future
end-user installations.

Revisit this record when the image pin, native profile/config CLI, role
selection, scanner implementation, or initializer locking/publication changes.

## Closeout checks

The final Phase 7 Docker acceptance run passed in 34.56 seconds, including a
Docker-client termination test: the native container process retained the same
writer lock until it finished, and a host installer could not acquire it early.
Native board/profile initialization, safe settings, credential non-cloning,
profile preservation and removal/restart checks passed in that run.

Unit tests, race tests, vet, formatting and local documentation links passed.
Independent review found two reporting gaps; regression tests reproduced them
before the fixes. Verification now reports absent Kanban as pending using only
file metadata, and plans disclose native initialization on existing deployments.
Concurrent generic-team work was outside this review and is not qualified by
this Phase 7 evidence.
