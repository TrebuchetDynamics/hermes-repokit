# Generic team shared memory qualification

Observed 2026-09-27. Configuration qualification only; model-backed acceptance
remains pending. No host credentials, provider calls, downloads or existing
services were used by these checks.

## Pinned artifacts and executed checks

- Hermes: `nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db`, OCI revision `749220ef0007f8d87bd1531f1c24b0fe93816385`.
- OpenViking: `ghcr.io/volcengine/openviking@sha256:569193efd49ad15a818c98ca66bfb566d1726713f1f3ec9c488b97fa66757d05`, OCI revision `3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1`, v0.4.21.

`go test ./internal/projectmemory` checks public configuration generation.
`REPOKIT_TEST_MEMORY_CONFIG_DOCKER=1 go test -count=1 -v ./internal/projectmemory`
executes the pinned images with `--pull=never --network none --rm`, no host
mounts and an explicit Python entrypoint. It passed for both images. These
probes do not start services or perform setup. The Hermes probe mocks the secret
and linked-config readers to exercise the real resolver; the six role labels
denote identical intended inputs, not five running profile sessions.

The probes verified:

- The public native memory section resolves to endpoint `http://openviking:1933`,
  account `repokit`, the supplied repository identity and an empty agent.
- Both Hermes built-in memory flags remain true alongside the external provider.
- A linked `actor_peer_id` or legacy `agent_id` becomes the effective agent.
  Omitting YAML `agent` alone therefore does not establish shared recall.
- Even an empty `OPENVIKING_USER` overrides the YAML user. Environment secrets
  and linked configuration must be checked through the active profile scope.
- OpenViking defaults `memory.extraction_enabled` to true.
- An empty OpenViking VLM configuration parses but reports unavailable. An empty
  embedding configuration fails because its model is missing.
- At this pin, an OpenAI-compatible embedding accepts `api_base` without an API
  key in its schema; an OpenAI VLM configured with model/base but no key fails
  credential validation. Schema acceptance is not endpoint compatibility.

The pinned server's `ApiKeyAuthPlugin.resolve_identity` was also inspected
inside the image. It obtains account/user from the API key manager and removes
client account/user assertion headers. This is source evidence of key-bound
identity, not a live cross-repository denial test.

## Public configuration contract

`internal/projectmemory.NativeConfig(repoID)` returns a new native Hermes
`memory` section with `provider: openviking`, `memory_enabled: true`,
`user_profile_enabled: true`, and `openviking.endpoint/account/user`. It has no
agent, peer or credential field, performs no I/O and does not activate anything.
The caller supplies the same stable deployment identity to all six profiles;
the current available identity is `target.Identity.Project`, captured in the
deployment's native artifacts. Repository relocation is not qualified here.

Before activation, native profile inspection must establish the same effective
endpoint/account/user for `default`, `researcher`, `planner`, `executor` and
`reviewer` and `steward`. `OPENVIKING_AGENT`, YAML `memory.openviking.agent`, and linked peer
configuration must be absent. No role-specific peer is created. The normal
project user key must resolve server-side to account `repokit` and this repo's
user; a root/admin key is not a substitute. Each repository needs its own
OpenViking service, durable storage and deployment network. Shared account
labels alone do not isolate account-level resource or agent scopes.

Native session sync/extraction is accepted for this deployment. Built-in memory
remains profile-local; it is not a shared replica or automatic OpenViking
failover. Neither successful health nor configuration parsing proves extraction.

## Missing private configuration for live acceptance

No private provider configuration was supplied to this qualification. Live
acceptance needs an owner-supplied native `ov.conf` containing:

| Configuration | Required concrete values |
| --- | --- |
| `embedding.dense` | Actual `provider`, `model`, provider endpoint `api_base`, matching vector `dimension` where required, correct `input` mode, and its credential/auth fields. For a custom endpoint, explicitly verify dimensions against returned vectors. |
| `vlm` | Actual `provider`, extraction-capable `model`, endpoint `api_base`, and backend-accepted credential/auth fields. At this pin `provider: openai` requires an API key even for the loopback-base schema check. Other providers have different requirements. |
| `storage.workspace` | `/opt/data/openviking/data`, backed by this repository's durable OpenViking directory. |
| `memory.extraction_enabled` | `true` (also the pinned default). |
| `server` | API-key authentication and a private root/bootstrap credential, used only to provision the account and normal repository user key. |
| Hermes native secret scope | A normal repository user key in the native mirrored connection under `/opt/data/.openviking`, linked by all six profiles without peer isolation. Conflicting `OPENVIKING_*` overrides must be reconciled in native setup. |

Provider-specific fields such as API version, custom headers or access-key pairs
must come from the actual chosen provider; no provider, model, private URL or
credential is invented by the contract. `OPENVIKING_API_KEY` authorizes Hermes
to OpenViking; it does not supply embedding or VLM provider authentication.
The container must be able to reach the selected model services. Local storage
does not imply local inference.

## Live acceptance still required

Use two disposable repository deployments A and B with separate services,
volumes, networks and normal user keys. Keep extraction enabled. Use unique,
non-sensitive markers and record native session/commit identifiers, extraction
completion, resulting memory URI and recalled content without logging secrets.

| Acceptance | Required evidence | Status |
| --- | --- | --- |
| Cross-profile recall | A default stores a durable fact using native memory/session extraction; fresh researcher, planner, executor, reviewer and steward sessions retrieve the same marker through native recall. Rotate writer roles to cover each role's write path. | Pending real model configuration |
| Effective identity | Native profile-scoped resolution and server user-key identity agree for every role, no peer source exists, built-in flags stay enabled. | Resolver behavior qualified; deployed profiles pending |
| Cross-repository isolation | B's normal recall does not return A's marker; direct read of A's resulting URI under B fails or is absent; A's user key is rejected by B's service. Include forged A account/user headers with B's key. | Pending live services/marker |
| Restart | Restart A's services, start fresh role sessions and repeat recall and B isolation. | Pending |
| Recreate | Remove/recreate containers while retaining native bind-mounted data, then repeat recall and B isolation. | Pending |
| RepoKit removal | Remove only the disposable RepoKit binary/source copy; use native launchers and ordinary Compose to repeat recall and isolation. | Pending |
| Outage/recovery | Stop A's OpenViking, verify local built-in memory is available and external failures are visible, restore service and confirm previously extracted memory; assess unsynced turns separately. | Pending |

Do not mark these rows passed from direct file insertion, fixed canned model
responses, mocked search, a ready endpoint or the configuration probes above.
Extraction can transform or decline a requested fact; `viking_remember` is not
an exact durable insert acknowledgment. A live result must include the actual
extracted memory and fresh-session recall evidence.

## Documentation provenance

Context7 resolved `/volcengine/openviking` and supplied current configuration and
identity documentation. Its moving-main examples were guidance only; the
executed assertions above use the pinned images. Relevant source references:
[Hermes native provider](https://github.com/NousResearch/hermes-agent/blob/749220ef0007f8d87bd1531f1c24b0fe93816385/plugins/memory/openviking/__init__.py),
[Hermes built-in flags](https://github.com/NousResearch/hermes-agent/blob/749220ef0007f8d87bd1531f1c24b0fe93816385/tools/memory_tool.py),
[OpenViking embedding schema](https://github.com/volcengine/OpenViking/blob/3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1/openviking_cli/utils/config/embedding_config.py),
[VLM schema](https://github.com/volcengine/OpenViking/blob/3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1/openviking_cli/utils/config/vlm_config.py),
[memory schema](https://github.com/volcengine/OpenViking/blob/3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1/openviking_cli/utils/config/memory_config.py),
[API-key identity](https://github.com/volcengine/OpenViking/blob/3fca2577520f00b7f580d85d4ac6ae42bb9ba6f1/openviking/server/auth/plugins/api_key.py).

## Production wiring follow-up

Normal install now embeds the pinned runtime inside Hermes with a private persistent directory;
`setup --memory` delegates native server/connection setup and links the shared
native connection across the roster. See [wiring evidence](openviking-wiring.md).
The original resolver probes above remain configuration evidence; live memory
rows are still pending.
