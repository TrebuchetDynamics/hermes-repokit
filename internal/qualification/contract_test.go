package qualification

import (
	"strings"
	"testing"
)

func TestContractRequiresVersionAndOperationSpecificFacts(t *testing.T) {
	good := Qualification{Supported: true, Source: "docs/qualification/hermes.md", Revision: strings.Repeat("a", 40), Contract: map[string]string{"image": "nousresearch/hermes-agent@sha256:" + strings.Repeat("b", 64), "executable": "hermes"}}
	if err := good.Require("image", "executable"); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Qualification){func(q *Qualification) { q.Supported = false }, func(q *Qualification) { q.Source = "" }, func(q *Qualification) { q.Revision = "main" }, func(q *Qualification) { delete(q.Contract, "image") }} {
		q := good
		q.Contract = map[string]string{"image": good.Contract["image"], "executable": "hermes"}
		mutate(&q)
		if q.Require("image", "executable") == nil {
			t.Fatal("accepted incomplete qualification")
		}
	}
}
func TestImmutableImageRejectsTagsAndMalformedDigests(t *testing.T) {
	for _, s := range []string{"x:latest", "x:v1", "@sha256:" + strings.Repeat("a", 64), "x@sha256:abc", "x@sha256:" + strings.Repeat("z", 64), "x\n@sha256:" + strings.Repeat("a", 64)} {
		if ImmutableImage(s) {
			t.Errorf("accepted %q", s)
		}
	}
	if !ImmutableImage("ghcr.io/org/image@sha256:" + strings.Repeat("a", 64)) {
		t.Fatal("valid pin rejected")
	}
}
