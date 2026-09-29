# Python import collision in Hermes workers

**Status:** upstream Hermes defect; RepoKit detects it and does not work around it.
Upstream: [NousResearch/hermes-agent#126127](https://github.com/NousResearch/hermes-agent/issues/126127)
(proposed fix [#126277](https://github.com/NousResearch/hermes-agent/pull/126277): spawn workers with `python -P`).
RepoKit's evidence was added as
[a comment](https://github.com/NousResearch/hermes-agent/issues/126127#issuecomment-5897286714).

## What happens

The pinned Hermes gateway starts Kanban workers as
`python3 -m hermes_cli.main -p <profile> ... chat -q "work kanban task <id>"`
with the repository mounted at `/workspace` as the working directory. `python -m`
puts the working directory first on `sys.path`, ahead of Hermes' editable
install. A repository-root module or regular package with the same top-level name
as a Hermes module is therefore imported instead of Hermes' own.

In a fresh RepoKit deployment of a real repository with a root `tools/` package,
the file-tool requirement check failed on every worker turn:

```
WARNING tools.registry: check_fn _check_file_reqs raised; dependent tools will be unavailable this turn
ImportError: cannot import name 'check_file_requirements' from 'tools' (/workspace/tools/__init__.py)
```

Hermes silently dropped `read_file`, `write_file`, `patch` and `search_files`. The
researcher correctly reported that it had no file tools and blocked; configuration
checks (`hermes tools list`, `verify`) all looked healthy.

## Deterministic reproduction (pinned image, no model call)

Same profile, same `--toolsets file,memory,web`, `HERMES_KANBAN_TASK` set:

| Launch | Working directory | Tools | File tools |
| --- | --- | --- | --- |
| `hermes ... prompt-size` (console script) | `/workspace` | 19 | present |
| `python3 -m hermes_cli.main ... prompt-size` | `/workspace` | 15 | **missing** |
| `python3 -m hermes_cli.main ... prompt-size` | `/` | 19 | present |

`sys.path` in the pinned runtime begins `['', ..., '__editable__.hermes_agent-0.21.5.finder.__path_hook__']`.

## RepoKit behavior

`verify` lists repository-root entries that Python would import in place of a
pinned Hermes module (`qualification.HermesImportNames`, derived from the pinned
image's worker entry points; requalify on every image change) and reports
`python-imports` degraded, which makes `CORE_TEAM` degraded. It follows Python
precedence: `name.py`, `name.pyc`, extension modules and `name/` directories with
`__init__` collide; nested paths, namespace directories without `__init__`, and
names Hermes does not import at runtime (for example `setup.py`) do not.

RepoKit never renames or modifies repository files and deliberately does not set
`PYTHONSAFEPATH` for the container: that would change the project's own Python
behavior (for example `python -m <local package>` from the repository root).
Remediation belongs upstream; an owner may alternatively rename the colliding
root module.

## Upstream report

The issue already existed (found through a workspace `hermes_cli/` directory causing
crashes). RepoKit's comment adds that ordinary root names such as `tools/` cause a
**silent** toolset loss rather than a crash, the no-model reproduction above, and
two suggestions: a `tools/__init__.py` regression case, and surfacing a raising
`check_fn` as an error.

When a Hermes release with the fix becomes the pinned `FoundationImage`, rerun the
reproduction; if workers no longer see the workspace on `sys.path`, the
`python-imports` probe and `HermesImportNames` can be retired.
