#!/bin/sh
# Build and install the RepoKit bootstrap CLI on this host.
set -eu

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then
    c_reset=$(printf '\033[0m')
    c_bold=$(printf '\033[1m')
    c_dim=$(printf '\033[2m')
    c_red=$(printf '\033[31m')
    c_green=$(printf '\033[32m')
    c_yellow=$(printf '\033[33m')
    c_blue=$(printf '\033[34m')
    c_cyan=$(printf '\033[36m')
else
    c_reset='' c_bold='' c_dim='' c_red='' c_green='' c_yellow='' c_blue='' c_cyan=''
fi

steps_total=5
step_no=0
title() { printf '%s\n' "${c_bold}${c_cyan}$*${c_reset}"; }
subtitle() { printf '%s\n' "${c_dim}$*${c_reset}"; }
step() { step_no=$((step_no + 1)); printf '\n%s\n' "${c_bold}${c_blue}[${step_no}/${steps_total}]${c_reset} ${c_bold}$*${c_reset}"; }
ok() { printf '%s\n' "  ${c_green}✔${c_reset} $*"; }
note() { printf '%s\n' "  ${c_dim}$*${c_reset}"; }
warn() { printf '%s\n' "${c_yellow}!${c_reset} $*" >&2; }
fail() { printf '%s\n' "${c_red}✗ repokit install: $1${c_reset}" >&2; exit 1; }

title 'Hermes RepoKit installer'
subtitle 'Builds the bootstrap CLI and installs it for this host user.'

step 'Checking host prerequisites'
[ "$#" -eq 0 ] || fail 'usage: install.sh (no arguments)'
[ "$(uname -s)" = Linux ] || fail 'the bootstrap CLI currently targets Linux'
command -v go >/dev/null 2>&1 || fail 'Go 1.26 or newer is required to build from source'
go_version=$(go version | awk '{print $3}')

case ${HOME:-} in
    /*) ;;
    *) fail 'HOME must be an absolute directory' ;;
esac
temp_root=$(CDPATH='' cd -- "${TMPDIR:-/tmp}" && pwd -P) || fail 'cannot use TMPDIR'
ok "Linux, ${go_version}"

step 'Resolving the RepoKit source'
# Run from a checkout when the script is a local file; otherwise assume it was
# fetched with `curl ... | sh`, where $0 is the shell rather than a readable file,
# and download the source tree to build from.
source_dir=
root=
if [ -f "$0" ]; then
    root=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd -P) || fail 'cannot locate source checkout'
fi
if [ -z "$root" ] || [ ! -f "$root/go.mod" ] || [ ! -d "$root/cmd/hermes-repokit" ]; then
    if [ -n "$root" ]; then
        fail 'install.sh must be run from a RepoKit source checkout'
    fi
    command -v curl >/dev/null 2>&1 || fail 'curl is required to download the RepoKit source'
    command -v tar >/dev/null 2>&1 || fail 'tar is required to unpack the RepoKit source'
    repokit_ref=${REPOKIT_REF:-main}
    repokit_url=${REPOKIT_SOURCE_URL:-https://github.com/TrebuchetDynamics/hermes-repokit/archive/refs/heads/$repokit_ref.tar.gz}
    source_dir=$(mktemp -d "$temp_root/repokit-source.XXXXXXXX") || fail 'cannot create source directory'
    archive=$source_dir/source.tar.gz
    note "Downloading RepoKit source from $repokit_url"
    curl -fsSL --retry 3 --connect-timeout 15 --max-time 300 -o "$archive" "$repokit_url" || fail "cannot download $repokit_url"
    tar -xzf "$archive" -C "$source_dir" || fail 'cannot unpack downloaded source'
    rm -f -- "$archive"
    root=$source_dir
    if [ ! -f "$root/go.mod" ]; then
        for candidate in "$source_dir"/*/; do
            if [ -f "$candidate/go.mod" ]; then
                root=$(CDPATH='' cd -- "$candidate" && pwd -P) || fail 'cannot locate downloaded source'
                break
            fi
        done
    fi
    [ -f "$root/go.mod" ] && [ -d "$root/cmd/hermes-repokit" ] || fail 'downloaded source is not a RepoKit checkout'
    ok "Downloaded and unpacked $repokit_ref"
else
    ok "Using checkout at $root"
fi

safe_directory() {
    [ -d "$1" ] && [ ! -L "$1" ] || fail "unsafe installation directory: $1"
    [ "$(stat -c %u -- "$1")" = "$(id -u)" ] || fail "installation directory is not owned by this user: $1"
    mode=$(stat -c %a -- "$1") || fail "cannot inspect installation directory: $1"
    case $mode in
        ''|*[!0-7]*) fail "invalid installation directory permissions: $1" ;;
    esac
    [ $((0$mode & 022)) -eq 0 ] || fail "installation directory is group/other writable: $1"
}

step 'Preparing the install directory'
safe_directory "$HOME"
for directory in "$HOME/.local" "$HOME/.local/bin"; do
    if [ ! -e "$directory" ] && [ ! -L "$directory" ]; then
        mkdir -m 755 -- "$directory" || fail "cannot create installation directory: $directory"
        note "Created $directory"
    fi
    safe_directory "$directory"
done
bin=$HOME/.local/bin
ok "Install target $bin"

step 'Building RepoKit bootstrap'
build_dir=$(mktemp -d "$temp_root/repokit-install.XXXXXXXX") || fail 'cannot create build directory'
tmp=
trap 'if [ -n "$tmp" ]; then rm -f -- "$tmp"; fi; if [ -n "$source_dir" ]; then rm -rf -- "$source_dir"; fi; rm -rf -- "$build_dir"' 0
(cd "$root" && TMPDIR="$temp_root" CGO_ENABLED=0 go build -trimpath -buildvcs=false -o "$build_dir/hermes-repokit" ./cmd/hermes-repokit) || fail 'Go build failed'
ok "Built static binary with ${go_version}"

step 'Publishing commands'
# Work relative to the checked directory, so a later path swap cannot redirect
# the final hard link into a different destination.
cd -P -- "$bin" || fail 'cannot enter installation directory'
safe_directory .
tmp=$(mktemp ./.repokit.XXXXXXXX) || fail 'cannot create temporary command'
cp -- "$build_dir/hermes-repokit" "$tmp" || fail 'cannot copy built command'
chmod 700 -- "$tmp" || fail 'cannot make built command executable'

# The program's own name is the canonical host command; `repokit` is a short
# alias. Generated repository launchers are `hermes-<repo>`, so for a repository
# whose name normalizes to `repokit` the `hermes-repokit` name is already a
# generated launcher. That launcher, and any other unowned command, is preserved.
owns_bootstrap() {
    # A RepoKit-built binary embeds this exact usage string.
    grep -aqF -- 'usage: hermes-repokit <plan|install|setup|verify>' "$1" 2>/dev/null
}

blocked=
preserved=
publish_command() {
    name=$1
    if [ -e "./$name" ] || [ -L "./$name" ]; then
        if [ ! -L "./$name" ] && [ -f "./$name" ]; then
            if cmp -s -- "$tmp" "./$name"; then
                note "Already installed: $bin/$name"
                return 0
            fi
            if owns_bootstrap "./$name"; then
                if ln -f -- "$tmp" "./$name" 2>/dev/null; then
                    ok "Updated RepoKit bootstrap: $bin/$name"
                else
                    warn "Blocked: cannot replace the RepoKit bootstrap at $bin/$name"
                    blocked=1
                fi
                return 0
            fi
        fi
        if [ -L "./$name" ]; then
            target=$(readlink -- "./$name" 2>/dev/null || :)
            case $target in
                */.hermes/bin/*)
                    warn "$bin/$name is a generated repository launcher -> $target; preserved"
                    note "Relocate it to install the bootstrap under this name, then rerun install.sh:"
                    note "  mv -- $bin/$name $bin/$name.launcher"
                    preserved=1
                    return 0
                    ;;
            esac
        fi
        warn "Blocked: existing command at $bin/$name differs; preserved unchanged. Remove or relocate it, then rerun install.sh"
        blocked=1
        return 0
    fi
    if ln -- "$tmp" "./$name" 2>/dev/null; then
        ok "Installed RepoKit bootstrap: $bin/$name"
    else
        warn "Blocked: cannot publish command at $bin/$name without replacing an existing entry"
        blocked=1
    fi
}

for name in hermes-repokit repokit; do
    publish_command "$name"
done

[ -z "$blocked" ] || fail 'one or more bootstrap names were preserved; see the messages above'

run_command=
for name in hermes-repokit repokit; do
    [ -f "$bin/$name" ] || continue
    owns_bootstrap "$bin/$name" || continue
    case :${PATH:-}: in
        *:"$bin":*)
            resolved=$(command -v "$name" 2>/dev/null || :)
            if [ "$resolved" = "$bin/$name" ]; then
                run_command=$name
                break
            fi
            warn "$resolved shadows $bin/$name on PATH; use the absolute command or reorder PATH."
            ;;
        *)
            warn "Add $bin to PATH, or run $bin/$name directly."
            run_command=$bin/$name
            break
            ;;
    esac
done
[ -n "$run_command" ] || fail 'no bootstrap command is installed; relocate the generated launcher and rerun install.sh'

printf '\n%s\n' "${c_green}${c_bold}RepoKit bootstrap installed.${c_reset}"
if [ -n "$preserved" ]; then
    note "A generated hermes-<repo> launcher kept its name; the bootstrap is available as ${run_command##*/}."
fi
title 'Next steps'
note 'Run these from the repository you want to prepare:'
printf '  %s plan\n' "$run_command"
printf '  %s install\n' "$run_command"
printf '  %s\n' '# run the printed Compose build/start command'
printf '  %s install   # after the runtime is running\n' "$run_command"
printf '  %s setup     # in your private terminal\n' "$run_command"
printf '  %s verify\n' "$run_command"
