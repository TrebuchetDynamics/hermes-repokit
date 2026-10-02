package verify

import (
	"os"
	"path/filepath"

	"github.com/TrebuchetDynamics/hermes-repokit/internal/boardwatch"
	"github.com/TrebuchetDynamics/hermes-repokit/internal/target"
)

// BoardWatch reports RepoKit's board watch job from host state. The owner's
// choice to pause, edit or remove it is reported, never treated as a failure.
func BoardWatch(id target.Identity) Probe {
	state, err := os.OpenRoot(filepath.Join(id.Root, ".hermes"))
	if err != nil {
		return Probe{"board-watch", Unknown, "native state unavailable"}
	}
	defer state.Close()
	job, marker, disk, err := boardwatch.Read(state)
	if err != nil {
		return Probe{"board-watch", Unknown, err.Error()}
	}
	deliver := "local"
	if job != nil {
		deliver = job.Spec.Deliver
	}
	switch boardwatch.Decide(job, marker, disk, boardwatch.Desired(deliver)) {
	case boardwatch.Current:
		return Probe{"board-watch", Healthy, "wakes default every 5 minutes only when the board is quiet with an open goal or stuck card"}
	case boardwatch.Upgrade:
		return Probe{"board-watch", Unknown, "an earlier release's board watch; install upgrades it"}
	case boardwatch.Paused:
		return Probe{"board-watch", Unknown, "owner paused the board watch job"}
	case boardwatch.OwnerModified:
		return Probe{"board-watch", Unknown, "owner changed the board watch job or script; RepoKit leaves it as is"}
	case boardwatch.OptedOut:
		return Probe{"board-watch", Unknown, "owner removed the board watch job; RepoKit leaves it off"}
	case boardwatch.OwnerJob:
		return Probe{"board-watch", Unknown, "a job named repokit-board-watch that RepoKit did not create"}
	}
	return Probe{"board-watch", Unknown, "not installed yet; install adds it once the team is configured"}
}
