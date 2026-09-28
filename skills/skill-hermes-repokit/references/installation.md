# Install or resume RepoKit

## Establish the target and preserve it

Resolve the user-specified path or `git rev-parse --show-toplevel`; change to
that root before every RepoKit command. Inspect repository instructions and:

```sh
git status --short --untracked-files=all
git branch --show-current
git rev-parse HEAD
git status --porcelain=v1 -z --untracked-files=all | sha256sum
```

Retain the baseline outside tracked source. For dirty files, record content
hashes as well: an unchanged status listing alone cannot prove preservation.
Do not reset, stash, clean, commit, or discard existing work for installation.

Check Docker/Compose availability and the selected local context. Inspect
existing `.hermes` ownership, tracked private-state paths, generated routing,
and matching container mounts without dumping credential files or Docker env.
An existing valid deployment is a resume, not a second installation.

## Obtain the bootstrap executable

Reuse a known RepoKit executable after checking its provenance and `--help`.
Otherwise use a verified local RepoKit source checkout. If none exists, clone
`https://github.com/TrebuchetDynamics/hermes-repokit.git` into a separate temporary
source directory, honor a requested revision, and record the checked-out commit.
Do not clone RepoKit over the target or update an existing checkout implicitly.

Build from that source root using its declared Go toolchain (`go.mod`; the
checked contract requires Go 1.26+). Use a fresh temporary directory for the
binary so an unrelated `/tmp/hermes-repokit` is not overwritten:

```sh
# Run in the verified RepoKit SOURCE checkout; stop on any failed check.
set -e
go test ./...
go test -race ./...
go vet ./...
repokit_gofmt_output=$(gofmt -l cmd internal tests)
test -z "$repokit_gofmt_output"
repokit_build_dir=$(mktemp -d "${TMPDIR:-/tmp}/repokit-bootstrap.XXXXXX")
go build -o "$repokit_build_dir/hermes-repokit" ./cmd/hermes-repokit
```

Carry that absolute binary path into subsequent commands and the private setup
handoff. Do not assume shell variables survive tool calls or exist in the
owner's terminal. Go is needed for source builds, not the generated runtime.
Missing Go/Docker or unsupported platforms are concrete prerequisites; do not
silently install host packages. The generated development image targets
Linux amd64; a different platform needs qualification, not guessed substitutions.

## Plan, publish, and start

From the **target root**, run the acquired binary's `plan`. Inspect its target,
container, project, launcher, Docker context, collisions, and unsupported items.
Refuse foreign/symlinked state, unknown generated-file edits, name collisions,
or unresolved ownership. Root-level Compose files can also be a current
installer collision; do not rename or bypass them to force installation.

If safe, run `install`. It publishes `.hermes/compose.yaml`, native defaults,
`.hermes/bin/hermes-<repo>`, embedded OpenViking storage scaffolding and
the pinned development-image recipe. It preserves recognized prior state and refuses
ambiguous changes. It does not start services.

Run the **exact all-service Compose build/start command printed by install**,
including context, absolute file, `--env-file /dev/null`, and selector cleanup.
This may download/build the pinned development image. Inspect the same project's
`compose ps`, then rerun `install` for native Kanban initialization and existing
team reconciliation. Do not handwrite a competing Compose deployment, change
memory provider or create a RepoKit runtime daemon.

## Private setup and resumption

Give the owner one concrete command, with actual quoted absolute paths:
`cd '<target-root>' && '<bootstrap-binary>' setup`.
The owner runs it in their private terminal; request only a completion signal,
not transcripts, screenshots, tokens, or keys. Keep the binary available until
bootstrap is complete. Do not use a captured PTY to conduct private setup.

Plain setup sequences native `default` setup, specialist cloning, native
OpenViking setup/linking and gated operational activation. Resume team reconciliation
with `setup --team`, or memory setup with `setup --memory`;
those flags are mutually exclusive. Preserve existing profiles, memories,
credentials, and owner choices on every rerun.

For OpenViking, follow the selected revision's `docs/bootstrap-quickstart.md`:
native server init/doctor; real owner-selected embedding and extraction/VLM
models; persistent `/app/.openviking/data`; private API-key authentication.
Use the native account `repokit` and normal repository user derived from `plan`.
OpenViking runs inside the Hermes container under native s6 supervision.
Hermes connects at `http://127.0.0.1:1933` using that user's key, not a root/admin
key, and no agent/peer partition. Existing native accounts/connections must be
inspected rather than recreated. Local Hermes memory remains enabled; native
OpenViking extraction/synchronization is intended behavior. Doctor may call the
configured providers; `verify` does not.

Successful setup requires native provider/profile/tool readiness and authenticated
shared memory before enabling the default gateway dispatcher and automatic review.
A real no-write researcher canary must pass. Active/finalizing workers defer setup;
queued work is preserved. Optional plugins remain owner-managed through native
admission. Respect scanner refusal and preserve existing plugin choices.

After setup, run `verify` and classify each component. A current result of
`review: unqualified` keeps its exit status nonzero even when other components
are healthy. Report that limitation without suppressing it or treating it as a
reason to reinstall everything.

## Optional persistent host command

Create a PATH link only when requested or already authorized. Prefer
`~/.local/bin/<plan-launcher-name>` if that directory is already on PATH.
Check `command -v`, `type -a`, filesystem entries including dangling symlinks,
and relevant shell aliases/functions. Preserve any conflicting command.
An existing exact link to this verified launcher can be reused.

Verify the target is the recognized executable launcher, the repository path
is stable, and link/target ancestors have trustworthy ownership and permissions.
Create a symlink with an exclusive operation (`ln -s`, never `ln -sf`), pointing
to the launcher's absolute path. Never link to the bootstrap binary.
If PATH or ownership is unsuitable, report the exact blocker and give the
absolute native launcher; do not edit shell rc files or PATH automatically.

Test command resolution in a fresh user shell, then `--help`, `kanban list`,
and `plugins list` from another directory. Bare chat requires successful private
model setup before it can be called usable. The link alone proves no chat or
integration acceptance.

Source contract: use the selected RepoKit checkout's `README.md` and `docs/bootstrap-quickstart.md`; match the installed CLI revision.
