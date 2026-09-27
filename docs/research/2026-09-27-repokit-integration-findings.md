> Historical source research. Policy conclusions about durable-only privacy and custom Nerve/Laya implementation are superseded by the final bootstrap directive. Source observations remain useful; native sync/extraction is now accepted.

# Hermes RepoKit integration findings

Date: 2026-09-27. **Source review, not runtime qualification or deployment approval.**

This is the concise evidence companion to the [bootstrap design](../superpowers/specs/2026-09-27-repokit-bootstrap-design.md). The research inspected upstream source and package/model metadata without installing dependencies, executing upstream code, downloading weights, running Docker or invoking inference. No deployed stack, image digest, model budget or capability boundary has been qualified.

## Research revisions versus release selections

| Component | Immutable research baseline | Deployment status |
| --- | --- | --- |
| Hermes | `NousResearch/hermes-agent@28e6496a5e3adfea57bebfc9571b981bff378523` | Source only; official image/platform/digest still requires selection and qualification. |
| OpenViking | `volcengine/OpenViking@a09a9d20a8e07d08973aee177802d00e08df29e6` | Official image repository `ghcr.io/volcengine/openviking` verified; no approved digest. Main project AGPL-3.0. |
| Laya released baseline R | `NandhaKishorM/laya@23a17522aa4942da6cce53a995a275760320b691`, released `laya==0.3.20` | Wheel SHA256 `6039e802fa5effb8dd492061cd7ad39a43087beadc4a4fa4a649614e77eb83d4`; four inspected core files match R. Not a qualified sidecar. |
| Laya newer source S | `NandhaKishorM/laya@4066d5d5fbf08b66c6757ddeedbd797bd7655bc0` | Still declares 0.3.20 but differs materially from R. S behavior must not be attributed to the released wheel. Apache-2.0. |
| Superpowers | Canonical `obra/superpowers`, native Hermes integration | No new approved deployment SHA selected here; require exact full SHA and scanner admission. |

## Hermes native surfaces and limits

- **Official layout and plugin admission:** the inspected [Dockerfile][h-docker] and [main wrapper][h-wrapper] support qualifying the official `/opt/data` layout rather than deriving an image to force HOME. [Native install source][h-plugin-install] and [scanner admission][h-scanner] ground exact-ref Superpowers installation and safe/caution/dangerous handling. They do not identify an approved plugin SHA or qualified image.
- **Profiles:** native `default` uses root home; named profiles use `profiles/`. Creation supports `--description`; `profile.yaml` holds description metadata. Decomposition reads descriptions; dispatch uses the actual profile selector. Creation must not silently clone personal credentials. [Profile creation][h-profile], [clone behavior][h-clone].
- **Capabilities:** effective `platform_toolsets.cli`, `agent.disabled_toolsets`, and worker `--toolsets` resolution matter. `file` includes both reads and writes; `safe` is not a read-only repository bundle. Custom toolsets and `pre_tool_call` veto are available, but RepoKit packaging and fail-closed policy are work to implement. Tool lookup failure can omit the explicit worker pin. Workers also auto-append Kanban lifecycle tools before disabled-toolset subtraction; merely omitting a selector does not deny completion. A mandatory loaded pre-tool policy is needed, but outer dispatch errors can continue execution and prechecks are not atomic DB enforcement. Tests/terminal/Python execute code under the shared UID; schema filtering is not an OS sandbox. [Worker auto-append][h-worker-tools], [outer hook-error path][h-hook-errors]. [Toolsets][h-tools], [worker resolution][h-worker], [tool expansion][h-expand], [pre-tool veto][h-veto].
- **Kanban:** `kanban_request_review` hands the same card to review; reviewer claim returns it to running; `kanban_request_changes` routes to recorded implementer; reviewer `kanban_complete` finishes. Native completion is broader than this flow and does not universally require independent review. Review/change transitions append durable events but do not emit the requested eight observer hooks at their transition sites. [Review transitions][h-review], [completion][h-complete].
- **Hooks:** synchronous, per-process, no durable broker or exactly-once guarantee. Claim/spawn and reclamation hooks can execute under dispatch flock; callbacks are exempt from normal callback timeouts. `on_kanban_dispatch_tick` is after lock release. Hook exceptions being caught does not make hung callbacks safe. Event payloads are versioned `hermes.observer.v1`; the dispatch result is a Python dataclass, not a JSON wire contract. [Emitter][h-emitter], [dispatch execution][h-dispatch], [callback behavior][h-callback], [timeout exclusions][h-timeout], [result shape][h-result].
- **Native OpenViking configuration:** `memory.provider: openviking`; `memory.openviking.endpoint/account/user/agent`; secret `OPENVIKING_API_KEY` via approved profile-scoped secret configuration, not YAML. Precedence is environment → linked ovcli → YAML → defaults. Optional agent means peer; leave it unset in every source for shared user recall. **0.2.10+ is recommended, not an enforced minimum.** [Schema][h-memory], [resolver][h-resolver], [provider documentation][h-memory-doc].

## OpenViking facts that change the design

1. **Blocker — raw session disclosure:** Hermes `sync_turn` uploads user/assistant text and non-recall tool inputs/results, then commits sessions for extraction. No exposed Hermes extraction/upload-disable setting was found. Sync runs as [background lifecycle work][h-background] outside normal tool calls, so hiding memory tools cannot stop it. The design preserves the exclusion by withholding incompatible memory-connected sessions/private processing pending a qualified native solution; an additional upload-opt-out enhancement needs owner scope approval and is not current support. This is not a blanket block on independent provisioning/non-inference observation. `viking_remember` submits an extraction source, not a guaranteed exact memory insert. Server `memory.extraction_enabled:false` stops extraction, not upload/archive; `session_auto_commit` is separate from Hermes explicit commits. Curated ingestion filters do not sanitize this path. [Turn sync][h-sync], [commit][h-commit], [server memory config][ov-memory].
2. **Key identity, not namespace labels, authorizes access:** use server API-key mode; key-bound account/user identity wins over headers. The configured client receives a project user key, not root/admin. This is not proof of root-key isolation: `.hermes/openviking` is also visible inside Hermes through the whole-home binds, so root/bootstrap secrets need separately verified permissions/execution boundaries. Account-level resource/agent scopes are shared, so retain per-repository service/data/network/key boundaries in addition to `repokit` / `repo_id` identity. [API-key plugin][ov-auth], [provisioning][ov-auth-doc], [namespace access][ov-namespace].
3. **Container:** `.hermes/openviking` → `/app/.openviking`; `ov.conf` is JSON, explicit `storage.workspace=/app/.openviking/data`. Local AGFS/vector backends avoid extra storage services. Default image starts VikingBot: explicitly set `OPENVIKING_WITH_BOT=0` or `--without-bot`. Do not copy upstream public-port/Caddy Compose. [Dockerfile][ov-docker], [entrypoint][ov-entry], [configuration][ov-config].
4. **Independent health:** unauthenticated `/health` proves neither client authorization nor successful embedding/extraction. `/ready` checks initialization/storage/auth/provider presence, not paid inference. Missing configuration can produce pending 503. [System routes][ov-health].
5. **No fake failover:** enabled Hermes built-in memory coexists in `$HERMES_HOME/memories/`, profile-local. External recall may return empty during outage. Pending-session recovery is not guaranteed replay of every unsent turn; local memory is not a shared OpenViking replica. [Initialization][h-init], [recall][h-recall], [local memory path][h-local].
6. **Privacy/cost:** embedding queries/content and VLM extraction context can leave the host according to `ov.conf`. Local storage does not mean local inference. Strict network isolation conflicts with remote providers unless an approved egress route exists. No model costs or sizing were measured. [Embedding calls][ov-embed], [extraction calls][ov-extract].

## Laya: actual versus proposed support

Canonical package is `laya[serve]`, command `laya-serve`; the distinct archived `stiermid/laya-serve` wrapper is not the same product. No qualified published official image was established. Source Docker packaging keeps PyTorch outside Hermes, uses UID/GID 10001 and `HF_HOME=/home/laya/.cache/huggingface`; HTTP startup must be explicit. [Package][l-package], [wrapper warning][l-wrapper], [Dockerfile][l-docker].

R serves `POST /v1/systemone` with `{state,questions,model?}` and `{model,answers,usage,routing}`. Optional `LAYA_API_KEY` gates inference; `/health` is unauthenticated and is not an inference/version attestation. R serializes inference without bounded admission; S adds admission limits and health revisions. Unknown model aliases can silently auto-route; preload names are not an allowlist. [R serving][l-serve-r], [S serving][l-serve-s], [Router][l-router].

`choice` yields labels/probabilities; `score` is expected ordinal index 0..K−1, so two levels give 0..1; `noul` is P(true), not boolean. Confidence is not correctness or control authority; no generated rationale is supplied. Router selects checkpoints, not Hermes profiles. [Typed semantics][l-agent].

**Blocker:** R's loader does not accept immutable model revisions. S adds revision/digest support but its stock server does not wire that configuration. A reviewed pin-aware sidecar bootstrap/Router injection or upstream fix is still required. Offline mode alone cannot pin an uncontrolled cache. Package/dependency locks, image provenance, checkpoint/config/tokenizer hashes, queue bounds and calibrated workloads remain qualification gates. [R loader][l-loader-r], [S revision constants][l-revisions], [S server construction][l-serve-s].

Research model pins (metadata inspected; weights not downloaded):

| Repository | Revision | Weights bytes |
| --- | --- | --- |
| `convaiinnovations/laya` | `55cf4c4ebb4ebe31b2550e8bdf3bd21b99753851` | 842609210 |
| `convaiinnovations/laya-multilingual` | `e4e9ddf21a7b1903b7acffd8814ad4307bf63a67` | 643835514 |
| `convaiinnovations/laya-typed-decisions` | `1a793eb568e6718f15941d08f85432581df534e3` | 842609220 |

[S revision constants][l-revisions] and the [pinned model card][l-card] identify these research candidates. CPU docs recommend 8 GB RAM/10 GB disk for quickstart, not a measured RepoKit minimum or three-model peak guarantee. GPU is optional; license, download and compute permission remain separate from package provenance. [Resource guidance][l-resources].

## Qualification boundary

Source evidence supports designing the integration. It does not prove live profile isolation, mandatory independent review, cross-profile recall, cross-repo denial, sidecar recovery, bounded inference or self-hosting. V1's design targets ordinary model-tool enforcement and concrete scoped execution guarantees using trusted selected runtime/admitted plugin implementations; it does not claim comprehensive resistance to hostile same-UID/plugin code. Repository tests are not trusted to obey role policy. Supported execution must deny protected-state mutation, source-writing reviewer actions and review bypass; unrestricted shell is not read-only, and a user-key configuration does not hide exposed root credentials.

Runner choice, policy failure/race handling, secret permissions, pin-aware packaging, candidate artifacts and qualification fixtures are engineering obligations within the approved topology, not a mandatory owner interview. Owner decisions concern actual privacy/enhancement scope, provider/data/compute authority and any proposed product/security compromise; deployment selections and scanner caution approvals remain explicit. Missing upload compatibility blocks affected memory sessions; missing execution/review guarantees blocks affected dispatch; missing compute authority blocks downloads/inference. Independent eligible infrastructure, observation and engineering qualification can proceed. Full stack and real self-hosting acceptance remain release gates, not claims established by source review.

[h-docker]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/Dockerfile
[h-wrapper]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/docker/main-wrapper.sh
[h-plugin-install]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins_cmd_install.py
[h-scanner]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins_cmd.py#L174-L210
[h-worker-tools]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/model_tools.py#L314-L340
[h-hook-errors]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/model_tools.py#L765-L800
[h-background]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/agent/memory_manager.py#L533-L556
[h-profile]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/profiles.py#L851-L939
[h-clone]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/profiles.py#L1284-L1300
[h-tools]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/toolsets.py#L105-L186
[h-worker]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/kanban_db_dispatch.py#L2645-L2735
[h-expand]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/model_tools.py#L281-L340
[h-veto]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins.py#L1945-L1965
[h-review]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/kanban_db.py#L3355-L3573
[h-complete]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/kanban_db.py#L2770-L2835
[h-emitter]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/kanban_db.py#L150-L271
[h-dispatch]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/kanban_db_dispatch.py#L1959-L2000
[h-callback]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins_dispatch.py#L209-L244
[h-timeout]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/plugins_dispatch.py#L26-L50
[h-result]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/hermes_cli/kanban_db_dispatch.py#L91-L155
[h-memory]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/plugins/memory/openviking/__init__.py#L53-L110
[h-resolver]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/plugins/memory/openviking/__init__.py#L744-L788
[h-memory-doc]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/plugins/memory/openviking/README.md#L5-L155
[h-sync]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/plugins/memory/openviking/__init__.py#L1961-L2055
[h-commit]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/plugins/memory/openviking/__init__.py#L2294-L2414
[h-init]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/agent/agent_init.py#L1286-L1364
[h-recall]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/plugins/memory/openviking/__init__.py#L1542-L1614
[h-local]: https://github.com/NousResearch/hermes-agent/blob/28e6496a5e3adfea57bebfc9571b981bff378523/tools/memory_tool.py#L39-L41
[ov-auth]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/openviking/server/auth/plugins/api_key.py#L68-L105
[ov-auth-doc]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/docs/en/guides/04-authentication.md#L46-L191
[ov-namespace]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/openviking/core/namespace.py#L317-L334
[ov-docker]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/Dockerfile#L109-L139
[ov-entry]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/deploy/docker/openviking-entrypoint.sh#L4-L135
[ov-config]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/docs/en/guides/01-configuration.md#L35-L66
[ov-health]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/openviking/server/routers/system.py#L56-L183
[ov-memory]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/openviking_cli/utils/config/memory_config.py#L12-L115
[ov-embed]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/openviking/models/embedder/openai_embedders.py#L366-L419
[ov-extract]: https://github.com/volcengine/OpenViking/blob/a09a9d20a8e07d08973aee177802d00e08df29e6/openviking/session/memory/extract_loop.py#L1165-L1191
[l-package]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/pyproject.toml#L5-L75
[l-wrapper]: https://github.com/stiermid/laya-serve/blob/b0bcbd53a8531d8c5688e3c9199f914e40cb96b4/README.md#L10-L39
[l-docker]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/Dockerfile#L1-L64
[l-serve-r]: https://github.com/NandhaKishorM/laya/blob/23a17522aa4942da6cce53a995a275760320b691/laya/serve.py#L162-L300
[l-serve-s]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/laya/serve.py#L54-L388
[l-router]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/laya/router.py#L45-L239
[l-agent]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/laya/agent.py#L565-L814
[l-loader-r]: https://github.com/NandhaKishorM/laya/blob/23a17522aa4942da6cce53a995a275760320b691/laya/agent.py#L211-L270
[l-revisions]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/laya/revisions.py#L1-L30
[l-card]: https://huggingface.co/convaiinnovations/laya/blob/55cf4c4ebb4ebe31b2550e8bdf3bd21b99753851/README.md#L1-L18
[l-resources]: https://github.com/NandhaKishorM/laya/blob/4066d5d5fbf08b66c6757ddeedbd797bd7655bc0/docs/docker.md#L3-L173
