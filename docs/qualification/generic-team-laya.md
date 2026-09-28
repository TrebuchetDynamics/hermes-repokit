# Generic team: actual local Laya qualification

Executed on Linux amd64, 2026-09-27. **Local typed inference and native Nerve
consumption passed in a disposable credential-free Hermes worker profile.**
This qualifies the narrow tuple below, not an authenticated agent turn, full
team collaboration, semantic decision accuracy, or the installer's production
sidecar lifecycle. No existing deployment or host credentials were used.

The newer [runtime integration record](runtime-integrations.md) supersedes this
experiment's deployment-pending status: pinned standalone packaging and all-six
native hooks/decisions with offline outage/removal/recreation are now exercised.
The mounted-environment procedure below remains historical evidence.

## Exact tuple

| Component | Tested identity |
|---|---|
| Hermes | `nousresearch/hermes-agent@sha256:d4da4a40cd7a28aba983775d9fd31d94cbf153eeb0cb9e844d6d0f612b7c24db` |
| Nerve catalog plugin | `nerve` 0.3.0, `b9e78dd5e00cf1117c563cada4d563a56f66e609` |
| Sidecar base | `python@sha256:f77ac9e44ae96ef2c90b8053ea08c31f8be030f824196b0ae4db6d462c84e51f` (Python 3.12 slim) |
| SDK | `laya==0.3.3`, `torch==2.8.0+cpu`, `transformers==4.57.6` |
| Model | `convaiinnovations/laya-typed-decisions`, revision `1a793eb568e6718f15941d08f85432581df534e3` |
| Weights SHA256 | `4fa56de72383a9d3efa9cfa78955733c81b9fc8067a587ca4beb82c78107a24e` |

The weights are 842,609,220 bytes / 421,293,830 parameters. The host had about
40 GiB available RAM, 129 GiB disk, and no NVIDIA GPU. The sidecar was limited
to four CPUs / 6 GiB memory; one idle measurement after loading was 1.856 GiB.
Dependencies lived in a fresh dedicated Python 3.12 environment mounted read-only
into the sidecar, outside Hermes' Python environment. All resolved versions are
in [laya-requirements.txt](../../internal/supervision/testdata/laya-requirements.txt).
These are version pins, not a complete wheel hash lock.

## Executed acceptance

1. Created a disposable `worker` profile in the exact Hermes image. Native
   `hermes -p worker plugins install nerve --no-enable` succeeded through the
   ordinary catalog admission path without force or scanner bypass. Git HEAD
   matched the catalog SHA, and native listing showed `not enabled`.
2. Downloaded the fixed model revision from the public model repository without
   authentication. Verified SHA256 for all five model/config/tokenizer files
   before loading and again after inference. The service loaded `/model`, a
   read-only bind of those files. `/model` is also the configured response model
   identity; HTTP model names alone are not checkpoint attestation.
3. Started the pinned Nerve sidecar on `127.0.0.1:8765`, sharing only the
   disposable Hermes container's network namespace. No ports, credentials,
   Docker socket or Hermes state were mounted into the sidecar. CPU thread
   count was four; Hub/Transformers offline flags were enabled.
4. Wrote native profile settings **before enabling**: `nerve_profile: lean`,
   `reflex_backend: laya`, `reflex_laya_base_url: http://127.0.0.1:8765`,
   `reflex_laya_model: /model`, `reflex_laya_timeout_seconds: 120`, under
   `plugins.entries.nerve.settings`. Disconnected the Docker bridge before
   `hermes -p worker plugins enable nerve` and subsequent plugin loading.
5. Loaded Nerve through Hermes' actual `discover_plugins()` and dispatched
   `nerve_stats` and `nerve_decide` through Hermes' native registry in the
   profile's nonroot runtime. The decision returned `billing`, provider `Laya`,
   transport `laya-local-http`, `LOCAL_ONLY`, `live_provider_call: false`, and
   receipt `jevrec-5b5b525f98fe43fdfa9b5dd6a4adacf7`. This was a loaded native
   plugin invocation, not just a standalone Nerve import.
6. From that same container, actual HTTP inference returned all three primitive
   types: choice `billing`, score `1.0215` on a 0–2 scale, and noul `0.671`.
   Probabilities and numeric bounds passed the checked contract.
7. Removed both disposable containers and started the checked-in offline
   Compose fixture using the retained state/model/dependencies. The native probe
   passed again with a new receipt and identical typed answers. No model download
   was possible: Hermes used `network_mode: none`, and Laya shared its namespace.
8. Stopped Laya deliberately. The loaded native decision tool returned
   `Laya sidecar connection failed` and `ok: false`; it did not switch providers.
   A subsequent whole-stack `up -d --force-recreate` also passed the native probe.
9. The final fixture with an exact-response healthcheck passed `up -d --wait
   --wait-timeout 120` and the native probe, producing receipt
   `jevrec-5554b28d767b85af5a37aa4498dfd840` in 185.045 ms. Scoped Compose
   teardown removed the disposable services; downloaded artifacts were retained
   only in the disposable qualification root for reproducibility.

The initial native decision took 179.79 ms, while the first recreated-container
native decision took 12,369.955 ms; the respective three-question HTTP calls
took 409.292 ms and 417.926 ms. These are individual observations, **not latency
guarantees or a savings benchmark**. The test's 120-second timeout was explicit;
the upstream five-second default was not qualified for this CPU tuple.

## Reproduction artifacts and ordering

- [laya_fetch_model.py](../../internal/supervision/testdata/laya_fetch_model.py)
  downloads only the five pinned public artifacts and verifies every SHA256.
  `--verify` checks an existing directory without downloading or modifying it.
- [laya_native_probe.py](../../internal/supervision/testdata/laya_native_probe.py)
  checks configuration and installed SHA, loads Hermes' native plugin registry,
  performs synthetic typed HTTP inference and a native Nerve decision, then
  validates the real receipt. `LAYA_EXPECT_UNAVAILABLE=1` instead checks a
  stopped-sidecar failure. It performs inference and writes receipts; it is not
  a read-only verification command.
- [laya-qualification.compose.yaml](../../internal/supervision/testdata/laya-qualification.compose.yaml)
  is the executed offline fixture. It is not installed automatically by repokit.
  Both services must be recreated together when Hermes' network namespace changes.

Prepare a **new disposable root**, selected as `LAYA_FIXTURE_ROOT`; do not point
the fixture at a real deployment. It needs `workspace/`, a mode-0700 `hermes/`,
the exact Nerve checkout in `nerve/`, the pinned model in `model/`, and a Python
3.12 venv in `venv/`. Install CPU torch from `https://download.pytorch.org/whl/cpu`
and the remaining pinned requirements from PyPI. The tested preparation used
`uv venv`, `uv pip install 'torch==2.8.0' --index-url
https://download.pytorch.org/whl/cpu`, then `uv pip install 'laya==0.3.3'
'transformers==4.57.6'`, always with `--python` targeting the new venv.

Run model preparation explicitly:

```sh
python3 internal/supervision/testdata/laya_fetch_model.py "$LAYA_FIXTURE_ROOT/model"
cp internal/supervision/testdata/laya_native_probe.py "$LAYA_FIXTURE_ROOT/workspace/"
```

Catalog admission needs a temporary networked Hermes container using only this
root's state/workspace. Use the pinned image, official entrypoint,
`HERMES_UID=1000`, `HERMES_GID=1000`, `HERMES_HOME=/opt/data`, and
`HERMES_SAFE_ROOTS=/workspace,/opt/data`, mounting `hermes/` at `/opt/data` and
`workspace/` at `/workspace`, with `sleep infinity`. Within it:

```sh
hermes profile create worker --no-alias --description 'Disposable local Laya qualification worker'
hermes -p worker plugins install nerve --no-enable
hermes -p worker config set plugins.entries.nerve.settings '{"nerve_profile":"lean","reflex_backend":"laya","reflex_laya_base_url":"http://127.0.0.1:8765","reflex_laya_model":"/model","reflex_laya_timeout_seconds":120}'
```

Stop/remove only that disposable container, start the offline Compose fixture,
wait for Laya's health endpoint from Hermes, and then enable Nerve. The fixture
has no external networking, so catalog installation cannot run inside it.
Use an isolated Compose project name; the executed project was
`repokit-laya-9lhb4o`.

```sh
docker compose -p repokit-laya-9lhb4o -f internal/supervision/testdata/laya-qualification.compose.yaml up -d --wait --wait-timeout 120
docker compose -p repokit-laya-9lhb4o -f internal/supervision/testdata/laya-qualification.compose.yaml exec -T hermes hermes -p worker plugins enable nerve
docker compose -p repokit-laya-9lhb4o -f internal/supervision/testdata/laya-qualification.compose.yaml exec -T -u 1000:1000 -e HOME=/opt/data -e HERMES_HOME=/opt/data/profiles/worker -w /opt/hermes hermes python /workspace/laya_native_probe.py
```

Repeat the probe after `up -d --force-recreate` for both services, after Laya is
healthy. For the failure case, stop only `laya` and invoke the probe with
`-e LAYA_EXPECT_UNAVAILABLE=1`. Cleanup is this exact fixture's Compose `down`;
retain or remove its disposable bind files separately. No shared volumes or
existing deployments are part of rollback.

## Findings and remaining boundaries

- At this catalog pin, Lean intentionally exposes `nerve_decide` but omits
  `nerve_assess`, `nerve_rank`, and `nerve_verify`. `nerve_stats` accepts
  `section: summary`; the skill's `section: reflex` instruction is unsupported.
  The actual native decision receipt provides the provider-routing evidence.
- The upstream generic error wrapper labels the stopped-sidecar error transport
  `openrouter-decisions` despite `live_provider_call: false`. This is misleading
  error metadata, not evidence of a hosted call: the container was offline and
  the actual error was a refused local connection.
- Python slim with numeric UID 1000 has no passwd entry. Torch's default cache
  path calls `getpass.getuser()` during import; `USER=laya` is required for this
  read-only, nonroot fixture. A writable temporary compiler-cache location is
  also explicit. No runtime source was patched.
- The original upstream one-choice smoke returned `WATCH` for repeated failure,
  with near-even probabilities. Valid typed output is not evidence that a
  supervisor made the right judgment. This suite does not qualify semantic
  accuracy, calibration, full Nerve hook behavior, or completion authority.
- Dependency versions and model bytes persist outside the containers, but the
  sidecar is not yet a self-contained derived image. Production installer
  selection, versioned image build, durable health/readiness integration and
  removal-first team acceptance remain separate work. The bare production
  installer must not enable Nerve just because this fixture passed.
- No authenticated main-model/chat turn, six-profile team acceptance, full same-card maker/reviewer handoff,
  or performance/token-cost comparison was executed by this qualification.

Source baseline: [upstream-nerve.md](upstream-nerve.md). Model artifact:
[immutable model tree](https://huggingface.co/convaiinnovations/laya-typed-decisions/tree/1a793eb568e6718f15941d08f85432581df534e3).
