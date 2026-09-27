package qualification

import "testing"

const syntheticRevision = "0123456789abcdef0123456789abcdef01234567"

func TestMissingEvidenceIsUnknown(t *testing.T) {
	if got := Evaluate("", Setup, Evidence{}); got != Unknown {
		t.Fatalf("missing evidence = %v", got)
	}
}

func TestSupportedNeedsMatchingSourceAndRuntimeEvidence(t *testing.T) {
	for _, operation := range []Operation{NativeChat, Setup, Profiles, Plugins, Kanban, OfficialExecShim, ReadOnlyProbes} {
		t.Run(string(operation), func(t *testing.T) {
			evidence := Evidence{
				Operation: operation, SelectedRevision: syntheticRevision, Verdict: Supported,
				SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md",
			}
			if got := Evaluate(syntheticRevision, operation, evidence); got != Supported {
				t.Fatalf("complete synthetic evidence = %v", got)
			}
		})
	}
}

func TestUnsupportedNeedsMatchingSourceEvidence(t *testing.T) {
	evidence := Evidence{
		Operation: Plugins, SelectedRevision: syntheticRevision, Verdict: Unsupported,
		SourceReference: "synthetic/source.md",
	}
	if got := Evaluate(syntheticRevision, Plugins, evidence); got != Unsupported {
		t.Fatalf("source-backed unsupported evidence = %v", got)
	}
}

func TestIncompleteOrContradictoryEvidenceIsUnknown(t *testing.T) {
	base := Evidence{
		Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Supported,
		SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md",
	}
	tests := []struct {
		name             string
		selectedRevision string
		operation        Operation
		evidence         Evidence
	}{
		{"missing operation", syntheticRevision, Setup, Evidence{SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"unknown selected operation", syntheticRevision, Operation("unknown"), base},
		{"unknown evidence operation", syntheticRevision, Setup, Evidence{Operation: Operation("unknown"), SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"matching unknown operation", syntheticRevision, Operation("unknown"), Evidence{Operation: Operation("unknown"), SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"operation mismatch", syntheticRevision, Profiles, base},
		{"revision mismatch", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Setup, base},
		{"version label", "v1.2.3", Setup, base},
		{"matching version label", "v1.2.3", Setup, Evidence{Operation: Setup, SelectedRevision: "v1.2.3", Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"matching short revision", "01234567", Setup, Evidence{Operation: Setup, SelectedRevision: "01234567", Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"matching uppercase revision", "ABCDEF0123456789ABCDEF0123456789ABCDEF01", Setup, Evidence{Operation: Setup, SelectedRevision: "ABCDEF0123456789ABCDEF0123456789ABCDEF01", Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"short revision", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: "01234567", Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"uppercase revision", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: "ABCDEF0123456789ABCDEF0123456789ABCDEF01", Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"unknown verdict", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Verdict(99), SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"unknown zero verdict", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, SourceReference: "synthetic/source.md", RuntimeReference: "synthetic/runtime.md"}},
		{"supported without source", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Supported, RuntimeReference: "synthetic/runtime.md"}},
		{"supported without runtime", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: "synthetic/source.md"}},
		{"unsupported without source", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Unsupported}},
		{"blank source", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: " ", RuntimeReference: "synthetic/runtime.md"}},
		{"unsafe source", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: "../secrets", RuntimeReference: "synthetic/runtime.md"}},
		{"unsafe runtime", syntheticRevision, Setup, Evidence{Operation: Setup, SelectedRevision: syntheticRevision, Verdict: Supported, SourceReference: "synthetic/source.md", RuntimeReference: "https://user:password@example.test/log?token=x"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Evaluate(tt.selectedRevision, tt.operation, tt.evidence); got != Unknown {
				t.Fatalf("Evaluate returned %v, want unknown", got)
			}
		})
	}
}
