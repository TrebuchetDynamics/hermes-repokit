# Credential-free runtime observations — 2026-09-27

These observations are deliberately narrower than release qualification.

## Hermes

Official multi-architecture image candidate:
`nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db`.
OCI revision: `749220ef0007f8d87bd1531f1c24b0fe93816385`.
Native `hermes --version`: v0.21.5 (2026.9.24). `hermes version` is invalid.

A new disposable container with `/workspace`, `/opt/data`, explicit HERMES_HOME/safe roots and HERMES_UID/GID=1000 started with `sleep infinity` through the official entrypoint. No provider credentials were supplied. Existing containers were not modified.

Actual successful commands: `hermes --help`, `hermes profile create --help`, `hermes profile describe --help`, `hermes plugins install --help`, `hermes kanban --help`, `hermes kanban init`, `hermes profile create builder --no-alias --description TEXT`. Profile creation reported no API keys; default and builder homes persisted on the host. Bare chat help documents bare invocation; actual authenticated chat remains untested.

`kanban init` creates `/opt/data/kanban.db`. Other Kanban commands may auto-initialize the database; verify must not call them. Native profile creation makes a 0755 profile directory beneath the private 0700 Hermes root, so child read/execute permissions alone must not make safe reruns impossible.

## Superpowers admission

Native command executed without a TTY or force:
`hermes plugins install obra/superpowers --ref 8ca22dba9a94f28898bbce59f2537ff4d87c747d --no-enable`.

Result: exit 1, CAUTION, 229 findings, plugin refused. Exact scanner output is [superpowers-8ca22dba-scan.txt](superpowers-8ca22dba-scan.txt). Findings include documentation/test references to persistence, environment access, supply-chain operations and fixture tokens. These are scanner findings, not a claim that the plugin is malicious. User approval of this exact SHA/report is required before admission. No `--force` was used.

## OpenViking

Official candidate:
`ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05`.
OCI version v0.4.21, revision `3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1`.
Image was pulled; setup, doctor and actual memory operation are not yet qualified.

## Outstanding gates

Exact image exec UID/HOME, native setup terminal behavior, scanner approval/loading, OpenViking setup/storage/authorization, Nerve catalog pin and Laya transport/checkpoint, actual typed inference, distinct same-card run actors, removal-first acceptance and dogfood remain unproved. No offline fixture or successful pull satisfies these gates.
