#!/bin/sh
# Build and install the RepoKit bootstrap CLI from this checkout.
set -eu

fail() {
    printf 'repokit install: %s\n' "$1" >&2
    exit 1
}

[ "$#" -eq 0 ] || fail 'usage: install.sh (no arguments)'
[ "$(uname -s)" = Linux ] || fail 'the bootstrap CLI currently targets Linux'
command -v go >/dev/null 2>&1 || fail 'Go 1.26 or newer is required to build from source'

case ${HOME:-} in
    /*) ;;
    *) fail 'HOME must be an absolute directory' ;;
esac

temp_root=$(CDPATH='' cd -- "${TMPDIR:-/tmp}" && pwd -P) || fail 'cannot use TMPDIR'

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
    printf 'Downloading RepoKit source from %s...\n' "$repokit_url"
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

safe_directory "$HOME"
for directory in "$HOME/.local" "$HOME/.local/bin"; do
    if [ ! -e "$directory" ] && [ ! -L "$directory" ]; then
        mkdir -m 755 -- "$directory" || fail "cannot create installation directory: $directory"
    fi
    safe_directory "$directory"
done

bin=$HOME/.local/bin
build_dir=$(mktemp -d "$temp_root/repokit-install.XXXXXXXX") || fail 'cannot create build directory'
tmp=
trap 'if [ -n "$tmp" ]; then rm -f -- "$tmp"; fi; if [ -n "$source_dir" ]; then rm -rf -- "$source_dir"; fi; rm -rf -- "$build_dir"' 0

printf 'Building RepoKit bootstrap from %s with %s...\n' "$root" "$(go version)"
(cd "$root" && TMPDIR="$temp_root" CGO_ENABLED=0 go build -trimpath -buildvcs=false -o "$build_dir/hermes-repokit" ./cmd/hermes-repokit) || fail 'Go build failed'

# Work relative to the checked directory, so a later path swap cannot redirect
# the final hard link into a different destination.
printf 'Checking installation targets in %s\n' "$bin"
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
publish_command() {
    name=$1
    if [ -e "./$name" ] || [ -L "./$name" ]; then
        if [ ! -L "./$name" ] && [ -f "./$name" ]; then
            if cmp -s -- "$tmp" "./$name"; then
                printf 'RepoKit bootstrap already installed: %s/%s\n' "$bin" "$name"
                return 0
            fi
            if owns_bootstrap "./$name"; then
                if ln -f -- "$tmp" "./$name" 2>/dev/null; then
                    printf 'Updated RepoKit bootstrap: %s/%s\n' "$bin" "$name"
                else
                    printf 'Blocked: cannot replace the RepoKit bootstrap at %s/%s\n' "$bin" "$name" >&2
                    blocked=1
                fi
                return 0
            fi
        fi
        if [ -L "./$name" ]; then
            target=$(readlink -- "./$name" 2>/dev/null || :)
            case $target in
                */.hermes/bin/*)
                    printf 'Blocked: %s/%s is a generated repository launcher -> %s\n' "$bin" "$name" "$target" >&2
                    printf 'Relocate it to install the bootstrap under this name, then rerun ./install.sh:\n  mv -- %s/%s %s/%s.launcher\n' "$bin" "$name" "$bin" "$name" >&2
                    blocked=1
                    return 0
                    ;;
            esac
        fi
        printf 'Blocked: existing command at %s/%s differs; preserved unchanged. Remove or relocate it, then rerun ./install.sh\n' "$bin" "$name" >&2
        blocked=1
        return 0
    fi
    if ln -- "$tmp" "./$name" 2>/dev/null; then
        printf 'Installed RepoKit bootstrap: %s/%s\n' "$bin" "$name"
    else
        printf 'Blocked: cannot publish command at %s/%s without replacing an existing entry\n' "$bin" "$name" >&2
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
    case :${PATH:-}: in
        *:"$bin":*)
            resolved=$(command -v "$name" 2>/dev/null || :)
            if [ "$resolved" = "$bin/$name" ]; then
                run_command=$name
                break
            fi
            printf 'Warning: %s shadows %s/%s on PATH; use the absolute command or reorder PATH.\n' "${resolved:-$name}" "$bin" "$name" >&2
            ;;
        *)
            printf 'Add %s to PATH, or run %s/%s directly.\n' "$bin" "$bin" "$name" >&2
            run_command=$bin/$name
            break
            ;;
    esac
done
[ -n "$run_command" ] || run_command=$bin/hermes-repokit
printf '\nNext, from the repository you want to prepare:\n'
printf '  %s plan\n  %s install\n' "$run_command" "$run_command"
printf 'Run the printed Compose start command, then rerun %s install and complete %s setup in your private terminal.\n' "$run_command" "$run_command"
