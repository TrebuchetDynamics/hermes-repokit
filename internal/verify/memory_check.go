package verify

// MemoryCheckPreflight reports the installer’s qualification boundary, not a
// runtime capability observation. No supported lifecycle currently guarantees
// canary isolation and complete cleanup. Keep this gate passive: initializing
// the pinned Hermes provider can recover and commit unrelated pending sessions.
// A future adapter must qualify those guarantees before enabling any mutation;
// health, tool presence, or a newer image version alone cannot satisfy the gate.
func MemoryCheckPreflight() []Probe {
	return []Probe{
		{"memory-check", Unsupported, "blocked before execution: no qualified Hermes-mediated lifecycle guarantees canary isolation, complete cleanup, and initialization without unrelated pending-session recovery; ordinary verify remains available for read-only health"},
		{"memory-write", Unqualified, "not attempted: isolated Hermes-mediated writes are not qualified"},
		{"memory-extraction", Unqualified, "not attempted: extraction may merge into existing memories without a complete canary output manifest"},
		{"memory-recall-same-profile", Unqualified, "not attempted: fresh-context recall requires a safely isolated canary"},
		{"memory-recall-cross-profile", Unqualified, "not attempted: second-profile recall requires a safely isolated canary"},
		{"memory-cleanup", Unqualified, "not exercised: no canary was written; complete deletion of source sessions and derived memory is not qualified"},
	}
}
