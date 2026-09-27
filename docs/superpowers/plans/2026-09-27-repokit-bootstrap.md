# RepoKit Bootstrap Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (selected) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap independent native Hermes environments with one Go executable, then cease to be required.

**Architecture:** `hermes-repokit` writes `.hermes/compose.yaml`, native configuration and a standalone POSIX launcher. Compose/Hermes own runtime behavior; native plugins and optional Laya are independently packaged artifacts, never installer callbacks.

**Tech Stack:** Go standard library (`flag`, `testing`, `os/exec`, `context`, `encoding/json`, `embed`, `text/template`), YAML output, optional JSON receipt, POSIX shell, Docker Compose. Go is build/development tooling only. Bootstrap requires Git inspection and Docker/Compose on the supported host; the generated launcher requires only POSIX shell and Docker/Compose. No host Python/venv/pip/Node/Hermes prerequisite. Python remains valid INSIDE native plugin/Laya artifacts; Go is not Hermes's plugin API.

**Spec:** [bootstrap design](../specs/2026-09-27-repokit-bootstrap-design.md).

**Status:** Go amendment authorized by the user and independently reviewed with no blocking findings. Execute **only Task 1** in this session, then stop. Local offline builds/tests and separate documentation/implementation commits are authorized. No Docker installation, downloads, credentials, sidecar/plugin implementation or Hermes runtime work is in Task 1. Tasks 2–12 remain planned; obsolete 29-task A–G plans must not execute.

## Global Constraints

- First supported release: Linux amd64/arm64, qualified local Docker/Compose. Cross-compilation proves no macOS/Windows runtime support.
- One Hermes container/project; exact container AND command `hermes-<full-normalized-repo>`, no suffix/hash/truncation fallback. Identical basenames collide and refuse, including stopped containers.
- `.hermes/compose.yaml` relative base: `..` → `/workspace`, `.` → `/opt/data`; no project-directory override/root forwarding Compose. `HERMES_HOME=/opt/data`, `HERMES_WRITE_SAFE_ROOT=/opt/data:/workspace`.
- Four installer conveniences: plan/install/setup/verify. Native launcher forwards all explicit argv unchanged; zeroargs native chat only, qualified mapping if needed. No manager/session supervisor/health gate/auto-start.
- Default-only SAFE dispatch false, auto_decompose false; explicitly selected engineering preset also dispatch false. Native administration stays usable without RepoKit.
- Owner files authoritative; receipt optional/removable, never runtime authority. No secret hashes, socket/control endpoints, privilege workaround, second Hermes, provider or board.
- Immutable artifacts; unsupported selected operations remain unsupported. No latest upgrades or silent privacy compromise. No host language runtime needed to run binary/launcher.

## Review Focus

- Path aliases, same-name stopped containers and root Compose collisions → refuse (Task 2).
- Interrupted publication, stale receipt and lock-holder child surviving parent → preserve state/refuse conflicting writes (Task 4).
- Quotes/dollars/unrelated cwd/TTY/native errors and command conflicts → transparent launcher (Tasks 5–6).
- Native review hook error/race/auto-appended tools → unsupported surface withheld, not fake enforcement (Task 8).
- Installer removal and absent build tools → native commands/state still work (Task 12).

## Files, types and evidence rules

All product files are future targets. Module path **`hermes-repokit`**; packages under `internal/` use directory names. `cmd/hermes-repokit/main.go` only wires CLI and dependencies. Use stdlib embedded YAML templates with JSON-quoted scalar values (readable YAML), no framework. `go.mod` records the implementation-qualified Go version; `go.sum` is added only if a reviewed, pinned compile-time dependency becomes necessary, not fabricated for a stdlib-only module. Task 1 records the installed development toolchain; Task 12 separately qualifies the patched release toolchain.

`internal/qualification` owns version-bound operation evidence (Task 1). The following shared installer/test values are introduced in `internal/contract/types.go` when their owning tasks first need them; they are not Task 1 scaffolding or a runtime manifest:
```go
package contract

type Observation struct {
	Component, Verdict, Detail string
	ObservedUnix               int64
}
type Receipt struct {
	Version int
	Project string
	Created []string
}
type Artifact struct {
	Name, Digest string
	Content      []byte
}
type BootstrapRequest struct {
	Root, DockerContext, BinDir            string
	HermesImage, OpenVikingImage, LayaImage string
	Engineering                           bool
}
type BootstrapResult struct {
	Created, Preserved, Blocked []string
}
```
Tasks 2/3/5 define their own typed inputs. Runner dependencies live in Task 6, not a global runtime service. For every future test/build step set `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; acquire any separately approved build dependencies outside default tests. All default tests are Go and offline: `go test ./...` must never launch Docker/network/models or host Python. Use fake runners, local filesystem and shell/fake-Docker fixtures. Native plugin tests run only in authorized containers. Runtime tests use `//go:build runtime` AND explicit fixture scope/authority validation; missing authority fails an invoked runtime suite, never skip-to-green release evidence.

Each Go fence is a test/core file fragment with its package/imports; merge same-package fragments into listed files. Core snippets are not complete upstream integrations. Red means missing symbol/behavior; green requires the named negative cases too. Task 1 defines qualification contracts and records pending evidence; subsequent operation-owning tasks must qualify exact native commands/image shim behavior/pins before enabling them, never guess them.

### Task 1: Go CLI and native operation qualification contracts

**Files:** Create `go.mod`, `cmd/hermes-repokit/main.go`, `internal/cli/cli.go`, `internal/cli/cli_test.go`, `internal/qualification/evidence.go`, `internal/qualification/evidence_test.go`, `docs/qualification/native-contract.md`.
**Consumes:** approved spec and existing research, not a running Hermes instance. **Produces:** `cli.Commands() []string`, `cli.Run(args []string, stdout, stderr io.Writer) int`; `qualification.Operation`, `Verdict`, `Evidence` and `Evaluate(selectedRevision string, operation Operation, evidence Evidence) Verdict`.

- [ ] Select the already-installed Go toolchain for offline development; record the actual version and patch limitations. Set the module's language floor from verified local compilation, not a fabricated release pin. No toolchain download; current patched release-toolchain qualification remains Task 12.
- [ ] Red: package-local table tests must reject extra host commands, flags and positional arguments; help succeeds and names exactly plan/install/setup/verify. Each recognized command without help returns exit 1 with an explicit not-implemented diagnostic, never a fake successful plan/install/setup/verify. Usage errors return 2; help returns 0. No arguments is usage error (native noarg chat belongs to the later generated launcher). Run `go test ./internal/cli ./internal/qualification` and observe missing behavior.
```go
package cli

import (
    "bytes"
    "testing"
)

func TestSkeletonDoesNotClaimInstallation(t *testing.T) {
    var out, diagnostics bytes.Buffer
    if code := Run([]string{"install"}, &out, &diagnostics); code != 1 || out.Len() != 0 || diagnostics.Len() == 0 {
        t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), diagnostics.String())
    }
}
```
- [ ] Green: use standard-library `flag.FlagSet` with `ContinueOnError` per command; `main` passes argv and streams and exits with `Run`'s code. No subprocesses, filesystem writes, config loading, runtime probes or hidden native command hierarchy. Reject input without echoing arbitrary argument values. Task 6 implements setup delegation; Task 11 wires actual handlers.
- [ ] Red/green qualification: define typed operation identifiers for native chat, setup, profiles, plugins, Kanban, official exec shim and read-only probes; these are evidence labels, **not native command spellings**. `Evidence` contains only operation, immutable selected Hermes revision, verdict and sanitized source/runtime evidence references. No credentials, environment, argv, raw logs or free-form native output fields. Verdict zero value is unknown; unsupported requires matching operation/revision plus source evidence; supported additionally requires runtime evidence. Missing, blank, unknown-enum, mismatched or source-only positive evidence evaluates unknown. Match a full 40-character lowercase commit SHA; arbitrary version labels cannot qualify. Tests use clearly synthetic references and never prove native behavior.
```go
package qualification

import "testing"

func TestMissingEvidenceIsUnknown(t *testing.T) {
    if got := Evaluate("", Setup, Evidence{}); got != Unknown {
        t.Fatalf("missing evidence = %v", got)
    }
}
```
- [ ] Record research references separately from deployment qualification. No selected release artifact or native runtime is qualified by this task; all operational support remains unknown until the selected version is actually qualified. Document pending exact command spellings, noarg chat mapping, UID/GID/HOME shim, safe startup and plugin scanner evidence. Do not invent aliases or run Hermes to populate fixtures. Evidence evaluation checks contract completeness, not reference authenticity or permission to execute.
- [ ] Full offline `go test ./...`, `go vet ./...`, and CGO-disabled local binary build pass with `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`. Independent implementation review; commit implementation separately from reviewed Go design/plan. Stop before Task 2.

### Task 2: Canonical target and collision refusal

**Files:** Create `internal/target/target.go`, `internal/target/target_test.go`.
**Consumes:** canonical root, qualified name limit and injected read-only inventory. **Produces:** `Identity{Root,Project,Name string}`, `Existing{Name,Project,Root string; Running bool}`, `Inspect(root string,containers []Existing) (Identity,error)`, `Normalize(base string, limit int) (string,error)`.

- [ ] Red: `go test ./internal/target`.
```go
package target

import "testing"

func TestName(t *testing.T) {
	got, err := Normalize("Polymarket-Mega-Bot", 255)
	if err != nil || got != "hermes-polymarket-mega-bot" {
		t.Fatal(got, err)
	}
	if _, err = Normalize("!!!", 255); err == nil {
		t.Fatal("empty accepted")
	}
}
```
- [ ] Green core (limit supplied by qualified platform contract, 255 above is fixture-only):
```go
package target

import (
	"fmt"
	"regexp"
	"strings"
)

func Normalize(base string, limit int) (string, error) {
	slug := strings.Trim(regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(strings.ToLower(base), "-"), "-")
	name := "hermes-" + slug
	if slug == "" || len(name) > limit {
		return "", fmt.Errorf("invalid repository name")
	}
	return name, nil
}
```
Derive stable project from canonical-path SHA256, separate from exact full container/command. Test aliases, control characters, empty/long names, same basenames/different projects, root Compose variants/overrides, dangling links/hardlinks, foreign/tracked `.hermes`, moved targets and stopped-container collisions. Task 6 runner later supplies actual container inventory; Task 2 tests inject it, never launch Docker. Existing artifacts inspectable, not automatically writable; no context switch/adoption/source chown.
- [ ] Same suite PASS. `git add internal/target && git commit -m "feat: reject ambiguous bootstrap targets"`.

### Task 3: Embedded readable Compose

**Files:** Create `internal/render/compose.go`, `internal/render/compose.yaml.tmpl`, `internal/render/compose_test.go`.
**Consumes:** target.Identity and qualified artifacts. **Produces:** `Input{Identity target.Identity; HermesImage,OpenVikingImage,LayaImage string}`, `Compose(Input)([]byte,error)`, `YAMLScalar(string) string`.

- [ ] Red: `go test ./internal/render`.
```go
package render

import (
	"hermes-repokit/internal/target"
	"strings"
	"testing"
)

func TestLayout(t *testing.T) {
	b, err := Compose(Input{Identity: target.Identity{Project: "fixture", Name: "hermes-fixture"}, HermesImage: "example.invalid/h@sha256:" + strings.Repeat("a", 64)})
	for _, s := range []string{"..:/workspace", ".:/opt/data", "hermes-fixture"} {
		if !strings.Contains(string(b), s) {
			t.Fatal(s)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}
```
- [ ] Green core:
```go
package render

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
	"text/template"
)

//go:embed *.tmpl
var templates embed.FS

func YAMLScalar(s string) string {
	b, _ := json.Marshal(strings.ReplaceAll(s, "$", "$$"))
	return string(b)
}
func Compose(in Input) ([]byte, error) {
	t, e := template.New("compose.yaml.tmpl").Funcs(template.FuncMap{"scalar": YAMLScalar}).ParseFS(templates, "compose.yaml.tmpl")
	if e != nil {
		return nil, e
	}
	var b bytes.Buffer
	e = t.Execute(&b, in)
	return b.Bytes(), e
}
```
Initial `compose.yaml.tmpl` core; extend only qualified selected sidecars:
```yaml
name: {{scalar .Identity.Project}}
services:
  hermes:
    image: {{scalar .HermesImage}}
    container_name: {{scalar .Identity.Name}}
    working_dir: /workspace
    environment:
      HERMES_HOME: /opt/data
      HERMES_WRITE_SAFE_ROOT: /opt/data:/workspace
    volumes:
      - "..:/workspace"
      - ".:/opt/data"
```
No service user override. Template mount strings above; sidecar sources `./openviking`, `./laya`. Validate immutable image syntax; optional Laya omitted. Test quotes/dollars, no installer source mounts, ports/socket/privileged/control endpoints, native `.env` excluded from interpolation and no project-directory override. Native artifact qualification, not fixture digest syntax, establishes deployability.
- [ ] Same suite PASS. `git add internal/render && git commit -m "feat: embed standalone Compose templates"`.

### Task 4: No-clobber publication and Linux-held locks

**Files:** Create `internal/install/publish.go`, `internal/install/lock_linux.go`, `internal/install/install_test.go`, `internal/install/lock_linux_test.go`.
**Consumes:** target identity, rendered artifacts, independent ownership proof (not receipt). **Produces:** `PublishNew(path string,data []byte,mode os.FileMode) error`, `Acquire(path string)(*os.File,error)`; installer-only `contract.Receipt` JSON.

- [ ] Red: `go test ./internal/install`.
```go
package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNoClobber(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte("owner"), 0600); err != nil {
		t.Fatal(err)
	}
	if PublishNew(p, []byte("stale"), 0600) == nil {
		t.Fatal("overwrite")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "owner" {
		t.Fatal(string(b))
	}
}
```
- [ ] Green publication core after verified private parent/leaf preflight:
```go
package install

import (
	"os"
	"path/filepath"
)

func PublishNew(path string, data []byte, mode os.FileMode) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".bootstrap-")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if e = f.Chmod(mode); e != nil {
		return e
	}
	if _, e = f.Write(data); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Link(f.Name(), path); e != nil {
		return e
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
```
Only 0600 native/private files or 0700 new launcher allowed; test modes and symlink/ancestor races using pinned directory identity. Implement Linux `syscall.Flock(fd, LOCK_EX|LOCK_NB)` on private regular lock; process lock stays held throughout transaction. Pass same open description via child `ExtraFiles` for mutating subprocesses; release by closing descriptors, **not LOCK_UN while child survives**. Test parent death with live child retaining lock, stale PID, competing installer, cleanup after last close. No portable-lock fiction or deleting lock to recover. Test interrupted link/fsync, partial effects/journal, failed pull/scanner, corrupt/absent receipt, unknown native files, ignored-but-indexed state and no-op rerun. Receipt never restores stale content or proves ownership.
- [ ] Same suite PASS. `git add internal/install && git commit -m "feat: preserve state with held Linux publication locks"`.

### Task 5: Standalone shell launcher and authorized shortcut

**Files:** Create `internal/launcher/launcher.go`, `internal/launcher/launcher.sh.tmpl`, `internal/launcher/launcher_test.go`, `internal/launcher/shortcut.go`, `internal/launcher/shortcut_test.go`; modify `internal/install/publish.go`.
**Consumes:** qualified native exec options and absolute Compose/context/name. **Produces:** `Config{Compose,Context,Executable string; ExecOptions,NoArgs []string}`, `Render(Config)([]byte,error)`, `Quote(string) string`, `Link(source,destination string)(bool,error)`.

- [ ] Red: `go test ./internal/launcher`; table-test fake Docker below. No real Docker.
```go
package launcher

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestForward(t *testing.T) {
	d := t.TempDir()
	docker := filepath.Join(d, "docker")
	if e := os.WriteFile(docker, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\nexit 23\n"), 0700); e != nil {
		t.Fatal(e)
	}
	c := Config{Compose: filepath.Join(d, "space $ quote'", ".hermes/compose.yaml"), Context: "fixture", Executable: "hermes"}
	b, e := Render(c)
	if e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(d, "hermes-fixture")
	if e = os.WriteFile(p, b, 0700); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(p, "unknown", "space $ quote'")
	cmd.Dir = "/"
	cmd.Env = append(os.Environ(), "PATH="+d+":"+os.Getenv("PATH"))
	out, e := cmd.CombinedOutput()
	if e == nil || cmd.ProcessState.ExitCode() != 23 {
		t.Fatal(e)
	}
	if !strings.HasSuffix(string(out), "hermes\nhermes\nunknown\nspace $ quote'\n") {
		t.Fatal(string(out))
	}
	if !strings.Contains(string(out), "exec\n-T\n") {
		t.Fatal(string(out))
	}
}
```
- [ ] Green quoting core (use `embed`/`text/template`, no runtime template files):
```go
package launcher

import (
	"bytes"
	"embed"
	"strings"
	"text/template"
)

//go:embed launcher.sh.tmpl
var shellTemplate embed.FS

func Quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
func Render(c Config) ([]byte, error) {
	t, e := template.New("launcher.sh.tmpl").Funcs(template.FuncMap{"quote": Quote}).ParseFS(shellTemplate, "launcher.sh.tmpl")
	if e != nil {
		return nil, e
	}
	var b bytes.Buffer
	e = t.Execute(&b, c)
	return b.Bytes(), e
}
```
Template with all list elements individually Quote-escaped; NoArgs only qualified empty→chat mapping, never explicit-argument rewriting:
```sh
#!/bin/sh
set -eu
unset COMPOSE_FILE COMPOSE_PROJECT_NAME COMPOSE_PROFILES COMPOSE_ENV_FILES
{{if .NoArgs}}if [ "$#" -eq 0 ]; then set -- {{range .NoArgs}}{{quote .}} {{end}}; fi
{{end}}set -- {{range .ExecOptions}}{{quote .}} {{end}}--workdir /workspace -e HERMES_HOME=/opt/data hermes {{quote .Executable}} "$@"
if ! [ -t 0 ] || ! [ -t 1 ]; then set -- -T "$@"; fi
exec docker --context {{quote .Context}} compose --env-file /dev/null -f {{quote .Compose}} exec "$@"
```
Expand fake-Docker Go tests to exact argv/context/file, all selector env cleared, noargs bare/mapped chat, setup/kanban/plugins/profile/gateway and unknown args, native errors/inherited streams. Test all four stdin/stdout TTY combinations with Linux PTY fixtures and Ctrl-C/process-group cleanup. No nonTTY help. No guessed UID/HOME or root-bootstrap-breaking user override; no health/receipt/session gate/start/pull/restart/eval/container shell.

Link tests preflight private regular executable/source/ancestors and destination, PATH/aliases/functions: identical verified link no-op, dangling/foreign link/EEXIST refusal, same basename collision no suffix. Unknown parent-shell resolution leaves PATH activation pending. Use `os.Symlink` without overwrite; no automatic rc/PATH/chmod changes, only new launcher 0700.
- [ ] Same suite PASS, `sh -n` generated fixtures. `git add internal/launcher internal/install/publish.go && git commit -m "feat: generate transparent native shell command"`.

### Task 6: Controlled processes and direct native setup

**Files:** Create `internal/host/process.go`, `internal/host/process_linux.go`, `internal/host/process_test.go`, `internal/cli/setup.go`, `internal/cli/setup_test.go`.
**Consumes:** Task 5 launcher. **Produces:** host `Invocation{Program string; Args,Env []string; Dir string; In io.Reader; Out,Err io.Writer; ExtraFiles []*os.File}`, `Runner interface{ Run(context.Context,Invocation)(int,error) }`, `ExecRunner{Env []string}` (explicit normalized environment, non-nil); cli `Setup(ctx context.Context,r host.Runner,launcher string,in io.Reader,out,stderr io.Writer)(int,error)`.

- [ ] Red: `go test ./internal/host ./internal/cli`.
```go
package host

import (
	"context"
	"testing"
)

func TestCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := (ExecRunner{}).Run(ctx, Invocation{Program: "must-not-start"}); e == nil {
		t.Fatal("started")
	}
}
```
- [ ] Green process core:
```go
package host

import (
	"context"
	"fmt"
	"os/exec"
)

type ExecRunner struct{ Env []string }

func (r ExecRunner) Run(ctx context.Context, in Invocation) (int, error) {
	if e := ctx.Err(); e != nil {
		return -1, e
	}
	c := exec.CommandContext(ctx, in.Program, in.Args...)
	c.Env = in.Env
	if c.Env == nil {
		c.Env = r.Env
	}
	if c.Env == nil {
		return -1, fmt.Errorf("explicit environment required")
	}
	c.Dir = in.Dir
	c.Stdin = in.In
	c.Stdout = in.Out
	c.Stderr = in.Err
	c.ExtraFiles = in.ExtraFiles
	e := c.Run()
	if c.ProcessState != nil {
		return c.ProcessState.ExitCode(), e
	}
	return -1, e
}
```
Complete Linux signal/cancellation handling: bounded installer probes terminate/reap their process group with context deadlines and bounded output; no orphan mutators. Interactive setup inherits TTY/streams, forwards Ctrl-C, no arbitrary probe deadline/capture. Test helper subprocesses, cancellation and inherited lock fd lifetime. Host Docker operations use direct argv (never `sh -c`), explicit context/file/env, no implicit context switch. Environment builder removes duplicate/selector variables, preserves only required qualified Docker/process environment; no secret logging.

`ExecRunner` environment builder must return a non-nil explicit slice (nil would inherit uncontrolled selectors). Setup core uses the concrete runner constructed with qualified normalized environment; nil environment is rejected, never silently inherited:
```go
package cli

import (
	"context"
	"hermes-repokit/internal/host"
	"io"
)

func Setup(ctx context.Context, r host.Runner, launcher string, in io.Reader, out, stderr io.Writer) (int, error) {
	return r.Run(ctx, host.Invocation{Program: launcher, Args: []string{"setup"}, In: in, Out: out, Err: stderr})
}
```
Setup delegates `[absoluteLauncher,"setup"]` with inherited private I/O; fake runner asserts exact args/exit/streams, no second wizard or validation hierarchy. Stopped service errors preserved; operator starts with explicit-file Compose, never launcher auto-start.
- [ ] Both suites PASS. `git add internal/host internal/cli/setup* && git commit -m "feat: delegate setup with controlled process IO"`.

### Task 7: Safe native presets and scanner admission

**Files:** Create `internal/native/preset.go`, `internal/native/preset_test.go`, `internal/native/config.yaml.tmpl`, `docs/native-operation.md`.
**Consumes:** qualification and artifact pins. **Produces:** `Preset{Profiles []string; Dispatch,AutoDecompose bool}`, `Engineering(bool) Preset`, `ScannerAllowed(verdict string,exactApproval bool) bool`.

- [ ] Red: `go test ./internal/native`.
```go
package native

import "testing"

func TestSafe(t *testing.T) {
	p := Engineering(false)
	if len(p.Profiles) != 1 || p.Dispatch || p.AutoDecompose {
		t.Fatal(p)
	}
	if ScannerAllowed("dangerous", true) {
		t.Fatal("scanner bypass")
	}
}
```
- [ ] Green core:
```go
package native

func Engineering(team bool) Preset {
	p := Preset{Profiles: []string{"default"}}
	if team {
		p.Profiles = append(p.Profiles, "researcher", "planner", "builder", "reviewer")
	}
	return p
}
func ScannerAllowed(v string, approval bool) bool { return v == "safe" || v == "caution" && approval }
```
Embed native config template. Test root default (not profiles/default), explicit preset/no credential clones, existing profile preservation, scanner absent/disabled/failed/stale caution rejection. Install independent pinned Nerve/review policy and full-SHA Superpowers through qualified native admission. Document native managed concurrency1/bounded fleet and safe downgrade retaining profiles/history; no admission receipt/default goal_mode.
- [ ] Same suite PASS. `git add internal/native docs/native-operation.md && git commit -m "feat: prepare safe native presets"`.

### Task 8: Independent native review policy qualification

**Files:** Create `plugins/review_policy/pyproject.toml`, `plugins/review_policy/review_policy/__init__.py`, `plugins/review_policy/tests/test_policy.py`, `internal/artifacts/review.go`, `internal/artifacts/review_test.go`, `docs/qualification/review-path.md`.
**Consumes:** trusted native same-card history/provenance. **Produces:** independently packaged Python `register(ctx)` policy; Go offline fixture `ReviewEvidence{BuilderProfile,ReviewerProfile,BuilderRun,ReviewerRun string; ExistingSameCard,DistinctActors,CurrentEvidence bool}`, `ValidReview(ReviewEvidence) bool` (test contract, NOT runtime guard).

- [ ] Red: `go test ./internal/artifacts`.
```go
package artifacts

import "testing"

func TestMissingReview(t *testing.T) {
	if ValidReview(ReviewEvidence{}) {
		t.Fatal("missing provenance")
	}
}
```
- [ ] Green offline predicate:
```go
package artifacts

func ValidReview(r ReviewEvidence) bool {
	return r.BuilderProfile == "builder" && r.ReviewerProfile == "reviewer" && r.BuilderRun != "" && r.ReviewerRun != "" && r.BuilderRun != r.ReviewerRun && r.ExistingSameCard && r.DistinctActors && r.CurrentEvidence
}
```
Native plugin binds evidence to actual records, current review-claimed run/candidate, not caller booleans. Container-only tests cover self/wrong/stale completion, changes/re-review, hook exceptions/races/auto-append, CLI/Python/SQLite bypass and protected writes via both aliases. Qualify scoped role runners; no controller or universal administrator enforcement. Unsupported path withholds affected dispatch. Declare and source-qualify native plugin discovery metadata in pyproject.toml for the distribution entry-point route, or include the native plugin.yaml/__init__.py layout if the selected installation route is directory-based. Test the actual admitted installed artifact and entry point, not only imports from a development source directory. Native plugin tests/import closure run without Go/installer source/receipt inside authorized Hermes tier; Go predicate passes prove no native enforcement.
- [ ] Offline suite PASS; record native qualification separately. `git add plugins/review_policy internal/artifacts/review* docs/qualification/review-path.md && git commit -m "feat: package independent native review policy"`.

### Task 9: Truthful OpenViking privacy gate

**Files:** Create `internal/memory/memory.go`, `internal/memory/memory_test.go`, `internal/memory/ov.conf.tmpl`, `docs/qualification/openviking.md`.
**Consumes:** source/runtime privacy and key-access evidence. **Produces:** `Supported(uploadCompatible,keyIsolated bool) bool`; embedded JSON server config template, no host credential schema.

- [ ] Red: `go test ./internal/memory`.
```go
package memory

import "testing"

func TestResearchPin(t *testing.T) {
	if Supported(false, true) || Supported(true, false) {
		t.Fatal("unsafe session enabled")
	}
}
```
- [ ] Green core:
```go
package memory

func Supported(uploadCompatible, keyIsolated bool) bool { return uploadCompatible && keyIsolated }
```
Refuse incompatible enabling BEFORE config writes/start; preserve existing bytes. Independent explicitly selected scaffold allowed, no provider substitution. Native build itself must prevent forbidden background uploads after installer removal; no pre-tool guard/invented opt-out. Upstream enhancement requires separate authority. Test precedence/shared profile identity/peer unset, root-key protection at actual UIDs and both aliases, repo-isolated keys/network/data, bot disabled/local storage, outage degradation not board stop. Actual memory write/recall/extraction and cross-repo denial are Task 12 authorized fixtures, not health/mock proof.
- [ ] Same suite PASS. `git add internal/memory docs/qualification/openviking.md && git commit -m "feat: refuse incompatible memory enabling"`.

### Task 10: Native Nerve and optional Laya artifact closure

**Files:** Create `plugins/nerve/pyproject.toml`, `plugins/nerve/nerve/__init__.py`, `plugins/nerve/tests/test_nerve.py`, `sidecars/laya/Dockerfile`, `sidecars/laya/requirements.lock`, `sidecars/laya/serve.py`, `sidecars/laya/tests/test_serve.py`, `internal/artifacts/laya.go`, `internal/artifacts/laya_test.go`, `docs/qualification/laya.md`.
**Consumes:** native event/HTTP contracts and immutable selections. **Produces:** standalone Python native `register(ctx)` observer/consumer; independently packaged optional server. Go `Pins{Image,Source,Dependencies,Checkpoint,Tokenizer,Config,Weights string}`, `Complete(Pins) bool` validates build-input completeness only.

- [ ] Red: `go test ./internal/artifacts`.
```go
package artifacts

import "testing"

func TestAbsentPins(t *testing.T) {
	if Complete(Pins{}) {
		t.Fatal("unpinned sidecar")
	}
}
```
- [ ] Green core:
```go
package artifacts

func Complete(p Pins) bool {
	return p.Image != "" && p.Source != "" && p.Dependencies != "" && p.Checkpoint != "" && p.Tokenizer != "" && p.Config != "" && p.Weights != ""
}
```
Also validate formats/actual hashes; no invented image/model pins. Go static closure/schema fixtures prohibit installer imports/receipt callbacks. Nerve native container tests: 4KiB whitelist envelopes, 256 ingress, nonblocking hooks under dispatch locks, one plugin-owned consumer, 16MiB spool/10,000 records or seven days, dedup/gaps/restart/read-only history; no board mutation or synchronous memory/inference. Optional Laya requires source-qualified pin-aware loader/admission (release/source drift remains), exact rubric/finite typed answers/model provenance, 16KiB caps, one in-flight/32 pending, 1s connect/10s total, one delayed retry/30s circuit. Outage/malformed/low confidence means UNKNOWN/WATCH. All Python tests container-only under scoped authority; no host pytest/pip or model downloads in default tests.
- [ ] Offline suite PASS; unqualified native artifacts remain unavailable. `git add plugins/nerve sidecars/laya internal/artifacts/laya* docs/qualification/laya.md && git commit -m "feat: isolate advisory runtime artifacts"`.

### Task 11: Bounded read-only plan/verify

**Files:** Create `internal/observe/observe.go`, `internal/observe/observe_test.go`, `internal/cli/install.go`, `internal/cli/install_test.go`; modify `internal/cli/cli.go`, `cmd/hermes-repokit/main.go`.
**Consumes:** host.Runner with read-only deadlines/output caps, actual files/services, not receipt desired state; Tasks 2–10 generation/admission primitives. **Produces:** `Probe func(context.Context)(contract.Observation,error)`, `Observe(ctx context.Context,p Probe) contract.Observation`, and one-shot `cli.Bootstrap(ctx context.Context, request contract.BootstrapRequest, runner host.Runner) (contract.BootstrapResult,error)`. Request fields are parsed installer input, never a runtime manifest.

- [ ] Red: `go test ./internal/observe ./internal/cli`.
```go
package observe

import (
	"context"
	"errors"
	"hermes-repokit/internal/contract"
	"testing"
)

func TestUnknown(t *testing.T) {
	got := Observe(context.Background(), func(context.Context) (contract.Observation, error) {
		return contract.Observation{Component: "laya"}, errors.New("private detail")
	})
	if got.Verdict != "unknown" || got.Detail != "" {
		t.Fatal(got)
	}
}
```
- [ ] Green core:
```go
package observe

import (
	"context"
	"hermes-repokit/internal/contract"
)

func Observe(ctx context.Context, p Probe) contract.Observation {
	o, e := p(ctx)
	if e != nil {
		o.Verdict = "unknown"
		o.Detail = ""
	}
	return o
}
```
Test file snapshots/no writes, no DB initialize/migrate, auth refresh, inference or dispatch dry-run; bounded subprocess groups/time/bytes/rows. Preserve freshness and per-component health/unsupported/privacy/coverage distinctions; sanitize diagnostics, no raw native output/secret hashes. Wire all four actual CLI handlers and exit statuses. Bootstrap calls target inspection, source-qualified artifact/privacy admission, held installer lock, render/no-clobber publication, authorized native default/plugin/preset initialization and launcher/link creation; it reports actual partial effects and exits. No timer, watch loop or runtime reconciliation. Install flags map to BootstrapRequest without arbitrary commands or executable paths. Use the qualified native argument arrays and host.Runner, not shell interpolation; recheck ownership/preimages before writes. Install-time container creation/start needs the invocation's scoped authority and exactly one intended Hermes service; it never captures setup or releases dispatch. An incompatible selected memory integration must refuse before any active provider configuration write/start. CLI integration fixtures invoke the real handlers against a fake runner and temporary targets: fresh default-only output + executable launcher, independently selected engineering preset still dispatch-off, receipt deletion without runtime dependency, no-op rerun, collision/pull/scanner failure and preserved owner edits. These test the installed command path, not only isolated helper predicates.
- [ ] Both suites PASS. `git add internal/observe internal/cli cmd/hermes-repokit/main.go && git commit -m "feat: wire one-shot bootstrap and read-only verification"`.

### Task 12: Reproducible binary and removal-first release

**Files:** Create `internal/release/release.go`, `internal/release/release_test.go`, `tests/runtime/bootstrap_test.go`, `tests/acceptance/bootstrap-independence.md`, `docs/bootstrap-quickstart.md`, `docs/build.md`.
**Consumes:** all prior tasks and explicit fixture authority. **Produces:** release evidence tiers and `Gate{Authorized bool; Fixture string}`, `Require(Gate) error` for explicitly invoked runtime tests.

- [ ] Red: `go test ./internal/release`.
```go
package release

import "testing"

func TestNoAuthority(t *testing.T) {
	if Require(Gate{Fixture: "disposable"}) == nil {
		t.Fatal("unauthorized")
	}
}
```
- [ ] Green core:
```go
package release

import "fmt"

func Require(g Gate) error {
	if !g.Authorized || g.Fixture == "" {
		return fmt.Errorf("runtime fixture authority required")
	}
	return nil
}
```
Runtime test file begins `//go:build runtime`; parse explicitly scoped fixture/data/budget/cleanup authority and call Require, fail missing authority, never skip as release pass. The separately authorized command is `go test -tags=runtime ./tests/runtime`; retain distinct non-inference and inference fixture selection and evidence. A no-Docker default test run cannot qualify this tier. Default offline suite uses no Docker/network/models. Build instructions after qualifying toolchain/module inputs: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=readonly -trimpath -buildvcs=false -ldflags='-s -w -buildid=' -o dist/hermes-repokit-linux-amd64 ./cmd/hermes-repokit`; repeat arm64 with matching output. Compare clean repeat-build bytes, archive toolchain/input/checksum provenance; test each architecture on actual authorized hosts. Embed all installer templates. No invented release hashes; standalone checksummed binary preferred, remote installer/curl-to-shell requirement absent, release script deferred.

Authorized disposable Linux host lacks Go/Python/Node/Pi for bootstrap; Git, Docker/Compose and shell remain. Bootstrap separate fresh target, explicit-context/file Compose up hermes, `hermes-<repo> setup`, then full up. Verify selected plugins/profiles/services. Remove installer binary/checkout/receipt from all import/subprocess/mount access, not just PATH. Invoke launcher from unrelated cwd with noargs/native commands and raw Docker/Compose commands; restart, actual bounded builder→distinct same-card reviewer task, restart again. Prove addressable sessions/logical history/actual selected memory/Nerve/selected cache persistence; credential comparisons private booleans only. Native Python INSIDE images remains legitimate.

Include interrupted bootstrap/stale child lock/collision/failed pull/scanner, owner edits, `down` without `-v`, optional conservative rerun, downgrade preserving profiles. Optional source dogfood has no self-apply/promotion. Privacy/runner/pin gaps block affected fixtures/full release; scoped inference requires enforced ceilings including extraction, time, tokens/spend/downloads/compute. Offline, credentialless runtime and actual inference evidence stay separate.
- [ ] Full offline `go test ./...` PASS; runtime/inference NOT RUN unless actually authorized/performed. `git add internal/release tests/runtime tests/acceptance/bootstrap-independence.md docs/{bootstrap-quickstart,build}.md && git commit -m "test: qualify standalone binary and installer-free runtime"`.

## Inline self-review

Twelve original boundaries retained. Go module/package paths and types match consumes/produces; snippets are default Go tests/core, native Python targets stay runtime-only and cannot depend on Go/installer. Go version/pins remain qualified selections, not fabricated artifacts. Task 4 owns Linux locking, Task 6 process cancellation, Task 5 shell transparency, Task 12 absent-build-tools/removal evidence. All review-focus failures have owning tests. No opaque runtime manifest, new control plane, latest update, host Python requirement or fake test evidence. Parent review also requires actual CLI bootstrap wiring in Task 11 and installed native plugin discovery tests, not just helper predicates. Independently review this Go amendment before Task 1; the user has authorized immediate Task 1 execution afterward without another architecture cycle.
