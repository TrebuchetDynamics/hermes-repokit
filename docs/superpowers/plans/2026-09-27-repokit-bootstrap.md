# RepoKit Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap an independently operable native Hermes repository environment, then cease to be required.

**Architecture:** A short-lived host CLI writes readable `.hermes/compose.yaml` and standalone native launcher and persistent native home artifacts. Hermes/Compose and independently packaged plugins own all runtime behavior; no installer receipt participates in execution.

**Tech Stack:** Proposed Python 3.11+ host CLI, pytest, PyYAML for readable YAML; Docker Compose, official Hermes, native plugins, OpenViking and optional Laya. Host/version floors and deployment artifacts require qualification, not assumptions from research pins.

**Spec:** [2026-09-27-repokit-bootstrap-design.md](../specs/2026-09-27-repokit-bootstrap-design.md).

**Status:** DRAFT replacement requested by user; review before implementation. The obsolete 29-task A–G plan must not execute. Selected execution method remains subagent-driven. This document authorizes no installs, network, credentials, Docker, downloads, inference or commits now. Commit instructions below apply only during later approved implementation.

## Global Constraints

- Exactly one Hermes runtime and one stable explicitly named Compose project per target repository.
- `HERMES_HOME=/opt/data` and `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`.
- Core CLI only `plan`, `install`, `setup`, `verify`; native administration always works without wrappers.
- Fresh default only; native dispatch false, `auto_decompose:false`; optional engineering preset also dispatch false.
- No resident manager, runtime receipt reads, RepoKit/Pi imports/mounts/calls, second Hermes, socket, control endpoint or privileged workaround.
- `.hermes/compose.yaml`, `..:/workspace` and `.:/opt/data` relative mounts (no project-directory override), immutable artifacts; never adopt/overwrite pre-existing root Compose.
- Owner runtime edits authoritative; preserve unknown native files and state; never publish secret hashes.
- Operation-specific qualification failures withhold affected surfaces only; selected unsupported features cannot be called working.

## Review Focus

- Alternate Compose filenames/overrides and symlink aliases: refuse ambiguous ownership (Task 2).
- Stale receipt plus owner-edited native files: never restore stale desired state (Task 4).
- Launcher quoting, TTY/defaults, stopped service and command collisions: preserve native argv/errors, no second Hermes or secret capture (Tasks 5–6).
- Native hook exception/race or worker tool auto-append: unsupported review path stays unavailable (Task 8).
- Installer import/subprocess/mount residue after removal: runtime still works, not just PATH hiding (Task 12).

## File map and execution rules

All product paths below are **future files**, not existing code. `src/repokit/` contains installer-only host code. `plugins/nerve/` and `plugins/review_policy/` are separately packaged native distributions without `repokit` imports; their installed copies are not symlinks. `sidecars/laya/` builds an independent artifact. Qualification Markdown files contain exact source-derived contracts and supported/unsupported operation evidence, never deployment pins invented from research. Tests use offline fixtures unless explicitly labeled authorized runtime/inference.

Tasks follow dependency order. Each test snippet names its imports locally; pytest's `tmp_path`/`monkeypatch` are standard fixtures. Ellipses are not implementation instructions. Core snippets specify the key algorithm, with remaining behavior enumerated as concrete tests. Qualification tasks yield executable fixture contracts before unsupported paths may be enabled. Do not substitute guessed upstream argv or settings to make a test green.

### Task 1: Establish native operation qualification and CLI boundary

**Files:** Create `pyproject.toml`, `src/repokit/__init__.py`, `src/repokit/cli.py`, `src/repokit/qualification.py`, `docs/qualification/native-contract.md`; test `tests/test_contract.py`.
**Interfaces:** Consumes research findings/spec. Produces `commands() -> tuple[str, ...]`, `require_supported(evidence: dict, operation: str) -> dict`; evidence entries have `supported: bool`, `source: str`, `contract: dict`. This is installer/package build input, never a runtime manifest.

- [ ] Red: add and run `pytest tests/test_contract.py -q`; expect missing-module failure.
```python
from repokit.cli import commands
from repokit.qualification import require_supported
import pytest

def test_only_bootstrap_commands():
    assert commands() == ('plan', 'install', 'setup', 'verify')
    with pytest.raises(ValueError):
        require_supported({}, 'profile_create')
```
- [ ] Green: package CLI entry point `hermes-repokit`; implement:
```python
def commands():
    return ('plan', 'install', 'setup', 'verify')

def require_supported(evidence, operation):
    item = evidence.get(operation, {})
    if item.get('supported') is not True:
        raise ValueError('unsupported operation: ' + operation)
    return item['contract']
```
Qualify official exec shim UID/GID/HOME and bare Hermes chat behavior (not currently verified); no guessed UID, forced Unix HOME or service-user override bypassing root bootstrap. Record exact source-qualified setup, profile creation/selection, plugin admission, Kanban config/history/probe and review mutation contracts at selected candidates. Distinguish facts from unqualified runtime behavior; native profile spelling cannot be inferred from aliases. Test missing/false evidence rejection. Unsupported OpenViking privacy and Laya packaging stay explicit. No network work without authority.
- [ ] Re-run same test (PASS); commit only listed files: `git add pyproject.toml src/repokit/{__init__,cli,qualification}.py docs/qualification/native-contract.md tests/test_contract.py && git commit -m "feat: define bootstrap-only contracts"`.

### Task 2: Inspect targets without adoption

**Files:** Create `src/repokit/target.py`, `tests/test_target.py`.
**Interfaces:** Consumes `Path`. Produces `inspect_target(root: Path) -> dict` with canonical `root`, stable canonical-path-hashed `project`, exact `container`/`command`, `collisions: list[str]`, `writable: bool`; no writes.

- [ ] Red: `pytest tests/test_target.py -q` fails before implementation.
```python
from repokit.target import inspect_target

def test_alternate_compose_collision(tmp_path):
    (tmp_path / 'compose.yaml').write_text('services: {}\n')
    before = sorted(tmp_path.iterdir())
    assert not inspect_target(tmp_path)['writable']
    assert sorted(tmp_path.iterdir()) == before
```
- [ ] Green core:
```python
names = ('compose.yaml', 'compose.yml', 'docker-compose.yaml',
         'docker-compose.yml', 'compose.override.yaml',
         'compose.override.yml', 'docker-compose.override.yaml',
         'docker-compose.override.yml')
collisions = [name for name in names if (root / name).exists()
              or (root / name).is_symlink()]
```
Derive a deterministic project identifier from canonical nonsecret repo identity, store it explicitly in Compose. Test relative/canonical aliases, dangling links, hardlinks, foreign ownership, relocation and tracked `.hermes` refusal. Container/command exactly `hermes-` plus full lowercase ASCII basename with nonalphanumeric runs replaced by hyphens and trimmed. Reject empty/invalid/over-qualified-length names, never truncate/default/suffix. Test same-basename different-path projects: different project IDs, identical container names, mandatory refusal including stopped foreign containers. No source chmod/chown. Existing generated targets are inspectable but not automatically writable.
- [ ] Same test suite PASS; `git add src/repokit/target.py tests/test_target.py && git commit -m "feat: inspect target ownership conservatively"`.

### Task 3: Render independent readable Compose

**Files:** Create `src/repokit/render.py`, `tests/test_render.py`.
**Interfaces:** Consumes qualified `images: dict[str,str]` (digest references), project/container strings and `laya: bool`. Produces `compose_document(project: str, container: str, images: dict, laya: bool=False) -> dict`, `render_yaml(document: dict) -> str`.

- [ ] Red: run `pytest tests/test_render.py -q` (missing module).
```python
from repokit.render import compose_document, render_yaml
import yaml

def test_native_layout():
    ref = 'example.invalid/hermes@sha256:' + 'a' * 64
    doc = compose_document('fixture', 'hermes-fixture', {'hermes': ref})
    service = doc['services']['hermes']
    assert service['volumes'] == ['..:/workspace', '.:/opt/data']
    assert doc['name'] == 'fixture'
    assert service['container_name'] == 'hermes-fixture'
    assert yaml.safe_load(render_yaml(doc)) == doc
```
- [ ] Green core:
```python
def render_yaml(document):
    import yaml
    return yaml.safe_dump(document, sort_keys=False)
```
Render official Hermes environment/safe roots, default service and only selected qualified sidecars; relative sidecar sources are `./openviking` and `./laya` from `.hermes`, not a second `.hermes` prefix. Reject mutable image tags, absolute/source-checkout mounts, socket/control endpoints, ports and privileged fields. Test digest rejection, optional Laya omitted, native `.hermes/.env` excluded from interpolation, explicit `--env-file /dev/null`, stable embedded project name and readable block YAML. Test official image entrypoint behavior against Task 1 evidence, not an invented daemon command.
- [ ] Same suite PASS; `git add src/repokit/render.py tests/test_render.py && git commit -m "feat: render standalone private compose"`.

### Task 4: Publish conservatively and preserve native state

**Files:** Create `src/repokit/install.py`, `tests/test_install.py`.
**Interfaces:** Consumes Task 2 inspection, Task 3 bytes and qualified native initialization/admission operations. Produces `publish_new(path: Path, content: bytes) -> None`, `install(root: Path, artifacts: dict[str, bytes], proof: dict) -> dict` reporting created/preserved/blocked paths. Optional `.hermes/repokit-install.json` is informational installer-only data, never authority.

- [ ] Red: `pytest tests/test_install.py -q`.
```python
from repokit.install import publish_new
import pytest

def test_owner_edit_wins(tmp_path):
    path = tmp_path / 'config.yaml'
    path.write_bytes(b'owner edit')
    with pytest.raises(FileExistsError):
        publish_new(path, b'stale receipt')
    assert path.read_bytes() == b'owner edit'
```
- [ ] Green core: publish staged private same-directory content without replacement:
```python
import os

def publish_new(path, content):
    import tempfile
    fd, temporary = tempfile.mkstemp(dir=path.parent)
    try:
        with os.fdopen(fd, 'wb') as stream:
            stream.write(content)
            stream.flush()
            os.fsync(stream.fileno())
        os.link(temporary, path)
    finally:
        os.unlink(temporary)
```
Wrap in held installer process lock, link/ancestor/ownership checks and directory fsync. Keep bounded transient failure journal; no runtime callback. Native initialization only on proven fresh home, dispatch disabled. Add interrupted-publication, surviving-child lock, corrupt/absent receipt, unknown native files, failed pull/scanner, ignored-but-indexed private state and no-op rerun tests. Rerun may add only independently proven owned missing artifacts; ambiguous proof refuses, no reset/adoption. Never auto-start inference or upgrade tags.
- [ ] Same suite PASS; `git add src/repokit/install.py tests/test_install.py && git commit -m "feat: publish bootstrap without overwriting native state"`.

### Task 5: Generate native launcher and collision-safe shortcut

**Files:** Create `src/repokit/launcher.py`, `tests/test_launcher.py`, `tests/test_shortcut.py`; modify `src/repokit/install.py`.
**Interfaces:** Consumes Tasks 1–4: qualified context/absolute Compose path, exec shim options, exact name, zeroarg native mapping. Produces `render_launcher(compose: Path, context: str, exec_options: list[str], executable: str, noargs: tuple[str,...]) -> str`, `link_launcher(source: Path, destination: Path) -> bool`. `noargs` is empty for qualified bare chat or the single qualified native chat argv; never derived from user arguments.

- [ ] Red: run `pytest tests/test_launcher.py tests/test_shortcut.py -q` before implementation (missing modules). Fake Docker records argv/TTY and exits 23, never contacts Docker:
```python
import os, pty, subprocess
import pytest
from repokit.launcher import render_launcher

@pytest.mark.parametrize('tty_in,tty_out', [(False,False),(True,False),
                                          (False,True),(True,True)])
@pytest.mark.parametrize('args', [[], ['setup'], ['kanban','list'],
    ['plugins','list'], ['profile','list'], ['gateway','status'],
    ['unknown', 'space $ quote\'']])
def test_passthrough(tmp_path, tty_in, tty_out, args):
    docker = tmp_path / 'docker'
    docker.write_text('#!/bin/sh\nprintf "%s\\0" "$@" > "$ARGV"\nexit 23\n')
    docker.chmod(0o700)
    launcher = tmp_path / 'hermes-fixture'
    compose = tmp_path / "space $ quote'" / '.hermes/compose.yaml'
    launcher.write_text(render_launcher(compose, 'fixture', [], 'hermes', ()))
    launcher.chmod(0o700)
    master, slave = pty.openpty()
    env = dict(os.environ, PATH=str(tmp_path)+':'+os.environ['PATH'],
               ARGV=str(tmp_path/'argv'), COMPOSE_FILE='wrong')
    try:
        result = subprocess.run([str(launcher), *args], cwd='/', env=env,
            stdin=slave if tty_in else subprocess.DEVNULL,
            stdout=slave if tty_out else subprocess.DEVNULL)
    finally:
        os.close(master); os.close(slave)
    actual = (tmp_path/'argv').read_bytes().split(b'\0')[:-1]
    expected = ['--context','fixture','compose','--env-file','/dev/null',
        '-f',str(compose),'exec'] + ([] if tty_in and tty_out else ['-T'])
    expected += ['--workdir','/workspace','-e','HERMES_HOME=/opt/data',
                 'hermes','hermes',*args]
    assert actual == [x.encode() for x in expected]
    assert result.returncode == 23
```
Add same test with qualified `noargs=('chat',)`: only empty args append `chat`, including pipes; explicit args unchanged. Fake Docker asserts all four selector variables unset and forwards stdin/stdout/stderr canaries. PTY process-group SIGINT fixture checks propagation/exit/no orphan; no claims about actual native sessions from fakes.

- [ ] Green shell-rendering core:
```python
from shlex import quote

def render_launcher(compose, context, exec_options, executable, noargs):
    mapping = ('if [ "$#" -eq 0 ]; then set -- ' +
               ' '.join(map(quote, noargs)) + '; fi\n') if noargs else ''
    prefix = ['docker', '--context', context, 'compose', '--env-file',
              '/dev/null', '-f', str(compose), 'exec']
    native = [*exec_options, '--workdir', '/workspace', '-e',
              'HERMES_HOME=/opt/data', 'hermes', executable]
    return ('#!/bin/sh\nset -eu\n'
        'unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES\n'
        + mapping + 'set -- ' + ' '.join(map(quote, native)) + ' "$@"\n'
        'if ! [ -t 0 ] || ! [ -t 1 ]; then set -- -T "$@"; fi\n'
        'exec ' + ' '.join(map(quote, prefix)) + ' "$@"\n')
```
Qualify `exec_options` against official shim; empty if shim safely drops user itself, explicit user/home only when demonstrated necessary/correct. No service `user` override or guessed HOME. No project override: embedded Compose name wins. Add absolute path/context checks at generation, not invocation; no eval/container shell, health/receipt/session callbacks or auto-start.

Add shortcut failure test before implementation:
```python
from repokit.launcher import link_launcher
import pytest

def test_dangling_shortcut_is_not_overwritten(tmp_path):
    source = tmp_path / 'source'
    source.write_text('#!/bin/sh\nexit 0\n')
    source.chmod(0o700)
    destination = tmp_path / 'hermes-fixture'
    destination.symlink_to(tmp_path / 'absent')
    with pytest.raises(FileExistsError):
        link_launcher(source, destination)
    assert destination.readlink() == tmp_path / 'absent'
```
Preflight source regular/private/executable, safe ancestors and all destination/PATH/shell alias/function conflicts before authorized same-name link. Unknown parent-shell resolution means installation pending, not collision-free. `os.symlink(source, destination)` never overwrites; verified identical link returns false, dangling/foreign link and EEXIST refuse. Test these cases, no suffix fallback, source permission preservation and no rc/PATH/chmod changes. Only newly generated launcher receives 0700. Same basename conflicts must fail even across different Compose projects.
- [ ] Both suites PASS plus `sh -n` generated scripts; `git add src/repokit/{launcher,install}.py tests/test_{launcher,shortcut}.py && git commit -m "feat: generate standalone native Hermes command"`.

### Task 6: Direct private native setup

**Files:** Create `src/repokit/setup.py`, `tests/test_setup.py`; modify `src/repokit/cli.py`.
**Interfaces:** Consumes Task 5 absolute generated launcher. Produces `setup(launcher: Path, run: Callable) -> int`, inherited-stream runner defaults to `subprocess.call`.

- [ ] Red: `pytest tests/test_setup.py -q` fails before implementation.
```python
from repokit.setup import setup

def test_setup_is_only_delegation(tmp_path):
    launcher = tmp_path / 'hermes-fixture'
    calls = []
    assert setup(launcher, lambda argv: calls.append(argv) or 17) == 17
    assert calls == [[str(launcher), 'setup']]
```
- [ ] Green core:
```python
def setup(launcher, run):
    return run([str(launcher), 'setup'])
```
No capture, separate wizard, secret args or specialist staging. Lead with `hermes-<repo> setup`; stopped-container native errors pass through. Document operator start using `docker --context CONTEXT compose --env-file /dev/null -f .hermes/compose.yaml up -d hermes` with selector env cleared; never auto-start. Full up is idempotent. Direct native Docker/Compose setup equally valid.
- [ ] Same suite PASS; `git add src/repokit/{setup,cli}.py tests/test_setup.py && git commit -m "feat: delegate private setup directly to native launcher"`.

### Task 7: Native presets and scanner-admitted plugins

**Files:** Create `src/repokit/native.py`, `tests/test_native.py`, `docs/native-operation.md`.
**Interfaces:** Consumes qualified exact native argv/config contracts and explicit preset selection. Produces `preset(engineering: bool=False) -> dict` with `profiles`, `dispatch`, `auto_decompose`; `scanner_allowed(verdict: str, exact_approval: bool) -> bool`.

- [ ] Red: `pytest tests/test_native.py -q`.
```python
from repokit.native import preset, scanner_allowed

def test_safe_presets():
    assert preset()['profiles'] == ['default']
    assert preset(True)['dispatch'] is False
    assert preset(True)['auto_decompose'] is False
    assert not scanner_allowed('dangerous', True)
```
- [ ] Green core:
```python
def preset(engineering=False):
    return dict(profiles=['default'] + (
        ['researcher', 'planner', 'builder', 'reviewer'] if engineering else []),
        dispatch=False, auto_decompose=False)

def scanner_allowed(verdict, exact_approval):
    return verdict == 'safe' or (verdict == 'caution' and exact_approval)
```
Map to qualified native config, never a RepoKit runtime schema. Test fresh root default (not profiles/default), explicit preset rerun preserving profiles/auth, descriptions and no credential cloning. Install immutable standalone Nerve/review-policy and full-SHA Superpowers through native admission; missing/disabled/failed scanners fail. Bind caution approval to exact current findings/tree; test stale approval rejection. Document native managed concurrency1, bounded fleet and safe downgrade preserving running/history/profile state, with no admission receipt or default goal mode.
- [ ] Same suite PASS; `git add src/repokit/native.py tests/test_native.py docs/native-operation.md && git commit -m "feat: prepare safe native profiles and plugins"`.

### Task 8: Qualify independent native review policy

**Files:** Create `plugins/review_policy/pyproject.toml`, `plugins/review_policy/review_policy/__init__.py`, `tests/test_review_policy.py`, `docs/qualification/review-path.md`.
**Interfaces:** Consumes trusted native history/actor/candidate projection, not worker prose. Produces standalone `review_policy.eligible(record: dict) -> bool`, native `register(ctx)` integration only for qualified supported paths. No RepoKit dependency or approval DB.

- [ ] Red: `pytest tests/test_review_policy.py -q`.
```python
from review_policy import eligible

def test_missing_provenance_fails_closed():
    assert not eligible({})
    assert not eligible(dict(builderProfile='builder', reviewerProfile='builder'))
```
- [ ] Green core:
```python
def eligible(r):
    return (r.get('builderProfile') == 'builder'
        and r.get('reviewerProfile') == 'reviewer'
        and bool(r.get('builderRun')) and bool(r.get('reviewerRun'))
        and r['builderRun'] != r['reviewerRun']
        and r.get('verifiedExistingSameCardRuns') is True
        and r.get('distinctActors') is True
        and r.get('currentCandidateEvidence') is True)
```
Qualification must bind booleans to native records and current review-claimed run, never trust caller assertions. Add wrong/missing run, stale candidate, self-completion, changes/re-review and current evidence fixtures. Exercise hook exceptions/races/tool auto-append and scoped runner attempts at source/credential/policy/DB writes through both aliases. Missing guarantees leave affected dispatch unsupported. Do not invent a new controller or claim helper predicates prove enforcement. Preserve native administrator APIs. Package/import in a subprocess with installer source unavailable.
- [ ] Same suite PASS; `git add plugins/review_policy tests/test_review_policy.py docs/qualification/review-path.md && git commit -m "feat: package qualified native review boundary"`.

### Task 9: Preserve truthful selected-memory qualification

**Files:** Create `src/repokit/memory.py`, `tests/test_memory.py`, `docs/qualification/openviking.md`.
**Interfaces:** Consumes native upload/authorization/permissions evidence. Produces `memory_session_supported(upload_compatible: bool, key_isolated: bool) -> bool`; renderer/installer report unsupported selection without connecting incompatible sessions.

- [ ] Red: `pytest tests/test_memory.py -q`.
```python
from repokit.memory import memory_session_supported

def test_research_pin_blocks_sessions_not_scaffold():
    assert not memory_session_supported(False, True)
    assert not memory_session_supported(True, False)
```
- [ ] Green core:
```python
def memory_session_supported(upload_compatible, key_isolated):
    return upload_compatible and key_isolated
```
Qualify native privacy behavior independent of receipt/CLI. No pre-tool privacy guard can stop background lifecycle sync. If incompatible, refuse enabling the selected integration before any active provider configuration write/start; leave existing state unchanged and report unsupported. Only an explicitly selected independent scaffold proceeds, with no silent provider substitution. Test rejection preserves existing config bytes and ordinary restart cannot activate the incompatible provider through newly generated config. Document separately authorized upstream enhancement, not local fake toggle. Test shared identity precedence/peer unset, per-repo keys/network/data, root-key denial at actual UIDs/aliases, bot disabled, local AGFS/vector configuration, outage degradation and no replacement provider. Authorized actual write/recall/extraction and cross-repo denial belong to Task 12, not mocked health claims.
- [ ] Same suite PASS; `git add src/repokit/memory.py tests/test_memory.py docs/qualification/openviking.md && git commit -m "feat: gate unsupported native memory sessions honestly"`.

### Task 10: Independent Nerve and optional bounded Laya

**Files:** Create `plugins/nerve/pyproject.toml`, `plugins/nerve/nerve/__init__.py`, `sidecars/laya/contract.md`, `sidecars/laya/Dockerfile`, `sidecars/laya/requirements.lock`, `sidecars/laya/serve.py`, `tests/test_nerve.py`, `tests/test_laya_contract.py`, `docs/qualification/laya.md`.
**Interfaces:** Consumes native whitelisted event dictionaries and optional fixed-rubric typed Laya answers. Produces standalone `nerve.offer(queue, event: dict) -> bool`, `nerve.unavailable() -> dict`, native `register(ctx)` tools/hooks; independent bounded consumer-owned derived state. Sidecar contract defines immutable build inputs, no installer runtime API. Selected sidecar produces `validate_pins(pins: dict) -> None` and an independently started HTTP server using source-qualified pin-aware loader injection; unselected/unqualified Laya is not built or deployed.

- [ ] Red: `pytest tests/test_nerve.py -q`.
```python
from queue import Queue
from nerve import offer, unavailable

def test_loss_never_blocks_dispatch():
    q = Queue(maxsize=1)
    assert offer(q, {'task': 'a'})
    assert not offer(q, {'task': 'b'})
    assert unavailable() == {'status': 'UNKNOWN', 'recommendation': 'WATCH'}
```
- [ ] Green core:
```python
from queue import Full

def offer(queue, event):
    try:
        queue.put_nowait(event)
        return True
    except Full:
        return False

def unavailable():
    return dict(status='UNKNOWN', recommendation='WATCH')
```
Add bounded 4KiB whitelist projection, 256-event ingress, single plugin-owned consumer, 16MiB spool/10,000-record or seven-day retention; dedup/gaps/restart/read-only-history fixtures. No lifecycle mutations or synchronous memory/model calls in hooks. Laya selection stays unavailable until pin-aware independent packaging qualifies release/source drift, image/dependency/model hashes and server compute admission. Contract tests: fixed four-question rubric, alias rejection, finite typed probabilities, malformed/oversized/stale responses, one in-flight call, 32 pending, 16KiB payloads, 1s connect/10s deadline, one delayed retry/30s circuit. Add an offline failing `tests/test_laya_contract.py` importing `serve.validate_pins` (test import path explicitly selects `sidecars/laya`), asserting `{}` raises `ValueError`; run `pytest tests/test_laya_contract.py -q` before implementation. Core:
```python
def validate_pins(pins):
    required = ('image_digest', 'source_revision', 'dependency_lock',
                'checkpoint_revision', 'tokenizer_hash', 'config_hash', 'weights_hash')
    if any(not pins.get(key) for key in required):
        raise ValueError('Incomplete immutable artifact identity')
```
Validate formats and actual acquired bytes, not merely key presence. Dockerfile accepts only qualified digest base input; requirements.lock binds exact approved dependencies/hashes. `serve.py` wires the selected version's loader revisions and bounded admission using Task 1/source findings, tested with fake loaders and no weights; stock unqualified serving is not a fallback. Source-specific wiring stays withheld until qualified, not guessed from release number. No model downloads now. Package isolation tests forbid RepoKit imports/receipt access; outage gives UNKNOWN/WATCH.
- [ ] Both suites PASS; `git add plugins/nerve sidecars/laya tests/test_nerve.py tests/test_laya_contract.py docs/qualification/laya.md && git commit -m "feat: isolate bounded native advisory artifacts"`.

### Task 11: Bounded actual-artifact plan and verify

**Files:** Create `src/repokit/observe.py`, `tests/test_observe.py`; modify `src/repokit/cli.py`.
**Interfaces:** Consumes target files and bounded read-only probe callbacks. Produces `observe(probes: dict[str, Callable]) -> dict`; `plan` combines observations with proposed actions, `verify` reports component freshness/unknowns without mutation.

- [ ] Red: `pytest tests/test_observe.py -q`.
```python
from repokit.observe import observe

def test_probe_unavailable_is_not_global_failure():
    def missing():
        raise TimeoutError()
    result = observe({'hermes': lambda: 'healthy', 'laya': missing})
    assert result['hermes'] == 'healthy'
    assert result['laya'] == 'unknown'
```
- [ ] Green core:
```python
def observe(probes):
    result = {}
    for name, probe in probes.items():
        try:
            result[name] = probe()
        except (TimeoutError, OSError):
            result[name] = 'unknown'
    return result
```
Every concrete probe has deadline/output/row bounds, redacted structured output and source-qualified nonmutating behavior. Test no credential refresh, inference, DB initialize/migrate or dispatch dry-run; compare files before/after, deny writes/subprocess mutation with fakes. Inspect actual Compose/native config, not receipt desired state. Report privacy/policy unsupported separately from healthy infrastructure, stale Nerve coverage, selected versus loaded Laya, memory auth versus health. Wire four argparse commands only.
- [ ] Same suite PASS; `git add src/repokit/{observe,cli}.py tests/test_observe.py && git commit -m "feat: inspect runtime without controlling it"`.

### Task 12: Removal-first release acceptance and preservation

**Files:** Create `tests/test_release_contract.py`, `tests/acceptance/bootstrap_independence.md`, `docs/bootstrap-quickstart.md`.
**Interfaces:** Consumes all prior artifacts plus explicit disposable-host/runtime/inference authorization. Produces redacted acceptance evidence classified offline/runtime/inference; no runtime manifest or improved-CLI promotion mechanism.

- [ ] Red: `pytest tests/test_release_contract.py -q` fails until acceptance recipe exists.
```python
from pathlib import Path

def test_removal_precedes_real_work():
    text = Path('tests/acceptance/bootstrap_independence.md').read_text()
    assert text.index('REMOVE_INSTALLER') < text.index('DISPATCH_REAL_TASK')
    assert '-f .hermes/compose.yaml restart' in text
    assert '-f .hermes/compose.yaml down' in text
```
- [ ] Green: write numbered executable operator recipe using qualified commands from Task 1; no guessed native aliases. Bootstrap a separate fresh target on authorized Pi-absent host. Show `docker --context CONTEXT compose --env-file /dev/null -f .hermes/compose.yaml up -d hermes`, then `hermes-<repo> setup`, then full up; clear selector environment. Use actual qualified context/name, not placeholders in handoff. Verify selected profiles/plugins/services. `REMOVE_INSTALLER`: remove checkout/binary/receipt and audit every mount/import/subprocess/resource closure, leaving generated shell/Compose/native state and independently installed plugins/sidecars. Invoke launcher from unrelated cwd (noargs chat and explicit native commands), then raw Docker/Compose native commands. Restart explicit-file Compose and perform native administration; verify exit/Ctrl-C behavior and native session coordination without a host supervisor. `DISPATCH_REAL_TASK`: actual bounded change, builder→distinct reviewer on same card, changes/re-review if needed. Restart again; verify addressable sessions/logical history/actual selected-memory recall/Nerve/cache persistence. Receipt absence must remain harmless.

Include optional source-improvement dogfood without self-apply/promotion. Test offline interrupted install/stale lock/collision/failed pull/scanner and owner-edit matrices; authorized runtime interrupted bootstrap, native down without `-v`, ordinary restart and optional conservative rerun. Native downgrade retains profiles. Credential checks emit booleans only. Missing privacy/runner/pin support blocks corresponding actual fixture and full release, not independent safe scaffold. Explicit ceilings cover inference/extraction/compute/downloads; no inferred permission. Runtime/inference recipes cannot run by default; mocks never satisfy them.
- [ ] Same test PASS; run full offline `pytest -q` and inspect dependency/artifact closure. Record actual runtime/inference as NOT RUN unless separately authorized and performed. `git add tests/test_release_contract.py tests/acceptance/bootstrap_independence.md docs/bootstrap-quickstart.md && git commit -m "test: require installer-free native runtime acceptance"`.

## Inline self-review

Coverage: ownership/layout Tasks 1–4; launcher/setup Tasks 5–6; profiles/review Tasks 7–8; memory Task 9; Nerve/Laya Task 10; observation Task 11; removal-first persistence Task 12. Each Review Focus item maps to a negative test. Latest amendment checked: mount base, exact collision-prone name, unchanged explicit argv, TTY/noargs semantics, standalone removal-first launcher, no inherited chat supervisor or manager. The precursor review records source-only limits. Interfaces use consistent names and primitive dictionaries; they are bootstrap/test contracts, not a new runtime schema. Snippets are core algorithms, not claims of complete enforcement. Qualification documents must carry exact version-specific integrations before release; unsupported operations fail explicitly rather than invent upstream commands/pins. No placeholders or execution approval implied. Review this draft before the preserved subagent-driven implementation method; no new questionnaire is required.
