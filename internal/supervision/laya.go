// Package supervision describes the locally qualified Nerve/Laya tuple.
// It does not deploy a sidecar, admit a plugin, or enable inference.
package supervision

const (
	NerveRevision     = "b9e78dd5e00cf1117c563cada4d563a56f66e609"
	LayaSidecarImage  = "python@sha256:f77ac9e44ae96ef2c90b8053ea08c31f8be030f824196b0ae4db6d462c84e51f"
	LayaModelRepo     = "convaiinnovations/laya-typed-decisions"
	LayaModelRevision = "1a793eb568e6718f15941d08f85432581df534e3"
	LayaWeightsSHA256 = "4fa56de72383a9d3efa9cfa78955733c81b9fc8067a587ca4beb82c78107a24e"
	LayaLocalEndpoint = "http://127.0.0.1:8765"
	LayaModelPath     = "/model"
)

// LocalLayaSettings returns a fresh settings object for
// plugins.entries.nerve.settings. Callers must verify the sidecar, model bytes,
// network namespace and plugin admission before enabling Nerve. /model is the
// sidecar's read-only pinned snapshot path, not a host or Hermes model path.
// The timeout reflects the qualified CPU fixture, not a latency guarantee.
func LocalLayaSettings() map[string]any {
	return map[string]any{
		"nerve_profile":               "lean",
		"reflex_backend":              "laya",
		"reflex_laya_base_url":        LayaLocalEndpoint,
		"reflex_laya_model":           LayaModelPath,
		"reflex_laya_timeout_seconds": 120,
	}
}
