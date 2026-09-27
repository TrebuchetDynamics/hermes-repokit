package native

import (
	"os"
	"strings"
	"testing"
)

func TestInteractiveInputRejectsNonTerminals(t *testing.T) {
	if InteractiveInput(strings.NewReader("")) {
		t.Fatal("reader accepted as terminal")
	}
	f, err := os.Open("/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if InteractiveInput(f) {
		t.Fatal("character device accepted as terminal")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if InteractiveInput(r) {
		t.Fatal("pipe accepted as terminal")
	}
}
