package plugins

import (
	"strings"
	"testing"
)

func TestAdmissionRequiresExactCurrentCautionApproval(t *testing.T) {
	sha := strings.Repeat("a", 40)
	scan := Scan{Revision: sha, Verdict: "caution", FindingsDigest: strings.Repeat("b", 64)}
	for _, a := range []Approval{{}, {Revision: strings.Repeat("c", 40), FindingsDigest: scan.FindingsDigest}, {Revision: sha, FindingsDigest: strings.Repeat("c", 64)}} {
		if Admit(scan, a) == nil {
			t.Fatal("accepted stale/missing approval")
		}
	}
	if e := Admit(scan, Approval{sha, scan.FindingsDigest}); e != nil {
		t.Fatal(e)
	}
	for _, v := range []string{"dangerous", "failure", "disabled", ""} {
		scan.Verdict = v
		if Admit(scan, Approval{sha, scan.FindingsDigest}) == nil {
			t.Fatalf("admitted %s", v)
		}
	}
	scan.Verdict = "safe"
	if e := Admit(scan, Approval{}); e != nil {
		t.Fatal(e)
	}
	scan.Revision = "main"
	if Admit(scan, Approval{}) == nil {
		t.Fatal("accepted moving revision")
	}
}
