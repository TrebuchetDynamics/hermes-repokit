# One-shot installer input. Never persisted as a runtime hook or launcher.
set -eu
workspace=$1
state=$2
[ "$(stat -c '%d:%i' "$workspace")" = "$3" ]
[ "$(stat -c '%d:%i' "$state")" = "$4" ]
[ "$(stat -c '%d:%i' "$workspace/.hermes")" = "$4" ]
[ ! -L "$workspace/.hermes-repokit.lock" ]
[ "$(stat -c '%d:%i' "$workspace/.hermes-repokit.lock")" = "$5" ]
[ -f "$state/config.yaml" ] && [ ! -L "$state/config.yaml" ]
export HERMES_HOME="$state"
unset HERMES_PROFILE HERMES_PROFILE_NAME HERMES_CONFIG HERMES_ENV

if [ -e "$state/kanban.db" ] || [ -L "$state/kanban.db" ]; then
    [ -f "$state/kanban.db" ] && [ ! -L "$state/kanban.db" ] && [ -s "$state/kanban.db" ]
else
    hermes kanban init >/dev/null
    [ -s "$state/kanban.db" ]
fi

# A nonempty file is not proof of a usable board. These native commands may
# initialize/migrate schema, so they belong in bootstrap, never read-only verify.
hermes kanban list --json >/dev/null
hermes kanban diagnostics --json >/dev/null

printf 'Native shared Kanban initialized; existing state preserved.\n'
