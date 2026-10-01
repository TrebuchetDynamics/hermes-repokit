// Package development describes the bounded development toolchain RepoKit builds
// into the Hermes image. Detection never executes repository-controlled code.
package development

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const maxManifestBytes = 1 << 20
const maxRootEntries = 4096
const maxManifests = 64

type Requirements struct {
	Go          bool     `json:"go"`
	Rust        bool     `json:"rust"`
	Flutter     bool     `json:"flutter"`
	Detected    []string `json:"detected"`
	Unsupported []string `json:"unsupported"`
}

// Detect inspects root manifests strictly and, within bounds, those of nested
// projects in a monorepo (rig-vigia/go.mod). Go, Rust and Flutter are
// provisioned;
// dependency installation, arbitrary version selectors and JVM provisioning
// remain unqualified.
func Detect(path string) (Requirements, error) {
	r := Requirements{Detected: []string{}, Unsupported: []string{}}
	root, err := os.OpenRoot(path)
	if err != nil {
		return r, err
	}
	defer root.Close()
	dir, err := root.Open(".")
	if err != nil {
		return r, err
	}
	entries, err := dir.ReadDir(maxRootEntries + 1)
	dir.Close()
	if err != nil && err != io.EOF {
		return r, err
	}
	if len(entries) > maxRootEntries {
		return r, fmt.Errorf("development detection: root exceeds %d entries", maxRootEntries)
	}
	found := map[string]bool{}
	manifestCount := 0
	for _, entry := range entries {
		name := entry.Name()
		kind := ""
		switch {
		case name == "go.mod":
			kind = "go"
		case name == "package.json":
			kind = "node"
		case name == "pyproject.toml" || (strings.HasPrefix(name, "requirements") && strings.HasSuffix(name, ".txt")):
			kind = "python"
		case name == "Makefile" || name == "makefile" || name == "GNUmakefile":
			kind = "make"
		case name == "Cargo.toml":
			kind = "rust"
		case name == "pubspec.yaml":
			kind = "flutter"
		case name == "pom.xml" || strings.HasPrefix(name, "build.gradle"):
			kind = "jvm"
		default:
			continue
		}
		manifestCount++
		if manifestCount > maxManifests {
			return r, fmt.Errorf("development detection: exceeds %d root manifests", maxManifests)
		}
		data, err := readManifest(root, name)
		if err != nil {
			return r, err
		}
		found[kind] = true
		switch kind {
		case "go":
			r.Go = true
			r.Unsupported = append(r.Unsupported, goRequirements(string(data))...)
		case "node":
			var pkg struct {
				Engines        map[string]string `json:"engines"`
				PackageManager string            `json:"packageManager"`
			}
			if len(strings.TrimSpace(string(data))) == 0 || strings.TrimSpace(string(data))[0] != '{' {
				return r, fmt.Errorf("development detection: package.json must be an object")
			}
			if err := json.Unmarshal(data, &pkg); err != nil {
				return r, fmt.Errorf("development detection: invalid package.json")
			}
			for _, engine := range []struct{ name, version string }{{"node", NodeVersion}, {"npm", NPMVersion}} {
				if constraint, ok := pkg.Engines[engine.name]; ok && !matchesVersion(engine.version, constraint) {
					r.Unsupported = append(r.Unsupported, "package.json "+engine.name+" version constraint not qualified by pinned runtime")
				}
			}
			for engine := range pkg.Engines {
				if engine != "node" && engine != "npm" {
					r.Unsupported = append(r.Unsupported, "package.json additional engine is not qualified")
				}
			}
			if pkg.PackageManager != "" && pkg.PackageManager != "npm@"+NPMVersion {
				r.Unsupported = append(r.Unsupported, "package.json packageManager is not the pinned npm")
			}
		case "python":
			if name == "pyproject.toml" {
				// Deliberately a narrow, fail-closed reader rather than an incomplete
				// TOML parser that claims all Python build systems are supported.
				for line := range strings.SplitSeq(string(data), "\n") {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "#") || !strings.Contains(line, "requires-python") {
						continue
					}
					m := pythonRequirement.FindStringSubmatch(line)
					if len(m) != 2 || !matchesPythonVersion(m[1]) {
						r.Unsupported = append(r.Unsupported, "pyproject.toml Python version constraint not qualified by pinned runtime")
					}
				}
			}
		case "rust":
			r.Rust = true
			r.Unsupported = append(r.Unsupported, rustRequirements(root, "", string(data))...)
		case "flutter":
			r.Flutter = true
			r.Unsupported = append(r.Unsupported, dartRequirements("", string(data))...)
		case "jvm":
			r.Unsupported = append(r.Unsupported, kind+" toolchain provisioning is not supported")
		}
	}
	detectNested(root, &r, found)
	for kind := range found {
		r.Detected = append(r.Detected, kind)
	}
	sort.Strings(r.Detected)
	sort.Strings(r.Unsupported)
	r.Unsupported = compact(r.Unsupported)
	return r, nil
}

func readManifest(root *os.Root, name string) ([]byte, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxManifestBytes {
		return nil, fmt.Errorf("development detection: %s must be a bounded regular file, not a symlink", name)
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, fmt.Errorf("development detection: %s changed while opening", name)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxManifestBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("development detection: %s exceeds size limit", name)
	}
	return data, nil
}

var pythonRequirement = regexp.MustCompile(`^requires-python\s*=\s*["']([^"']+)["']\s*(?:#.*)?$`)
var version = regexp.MustCompile(`^(?:v)?([0-9]+)(?:\.([0-9]+))?(?:\.([0-9]+))?$`)
var selector = regexp.MustCompile(`^(>=|<=|>|<|==|=|!=|\^|~)?\s*(v?[0-9]+(?:\.[0-9]+){0,2})$`)
var operatorSpace = regexp.MustCompile(`(>=|<=|>|<|==|=|!=|\^|~)\s+`)
var pythonSelector = regexp.MustCompile(`^(>=|<=|>|<|==|!=)([0-9]+\.[0-9]+(?:\.[0-9]+)?)$`)

func matchesPythonVersion(constraint string) bool {
	var normalized []string
	for _, part := range strings.Split(constraint, ",") {
		part = operatorSpace.ReplaceAllString(strings.TrimSpace(part), "$1")
		m := pythonSelector.FindStringSubmatch(part)
		if len(m) != 3 {
			return false
		}
		v := m[2]
		if strings.Count(v, ".") == 1 {
			v += ".0"
		}
		normalized = append(normalized, m[1]+v)
	}
	return matchesVersion(PythonVersion, strings.Join(normalized, " "))
}

func numbers(s string) ([3]int, bool) {
	var v [3]int
	m := version.FindStringSubmatch(s)
	if len(m) == 0 {
		return v, false
	}
	for i := range v {
		if m[i+1] == "" {
			continue
		}
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return v, false
		}
		v[i] = n
	}
	return v, true
}

func compare(a, b [3]int) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// matchesVersion recognizes simple ANDed numeric constraints. Anything else
// stays explicitly unqualified instead of silently accepting an incompatible pin.
func matchesVersion(actual, constraint string) bool {
	a, ok := numbers(actual)
	if !ok || strings.TrimSpace(constraint) == "" {
		return false
	}
	constraint = strings.ReplaceAll(constraint, ",", " ")
	// Join the optional whitespace between operator and version before splitting.
	constraint = operatorSpace.ReplaceAllString(constraint, "$1")
	for _, part := range strings.Fields(constraint) {
		m := selector.FindStringSubmatch(part)
		if len(m) == 0 {
			return false
		}
		v, ok := numbers(m[2])
		if !ok {
			return false
		}
		c := compare(a, v)
		switch m[1] {
		case ">=":
			if c < 0 {
				return false
			}
		case ">":
			if c <= 0 {
				return false
			}
		case "<=":
			if c > 0 {
				return false
			}
		case "<":
			if c >= 0 {
				return false
			}
		case "!=":
			if c == 0 {
				return false
			}
		case "^":
			upper := [3]int{v[0] + 1, 0, 0}
			if v[0] == 0 {
				upper = [3]int{0, v[1] + 1, 0}
				if v[1] == 0 {
					upper = [3]int{0, 0, v[2] + 1}
				}
			}
			if c < 0 || compare(a, upper) >= 0 {
				return false
			}
		case "~":
			if c < 0 || compare(a, [3]int{v[0], v[1] + 1, 0}) >= 0 {
				return false
			}
		default:
			parts := strings.Count(m[2], ".") + 1
			for i := 0; i < parts; i++ {
				if a[i] != v[i] {
					return false
				}
			}
		}
	}
	return true
}

func goRequirements(data string) []string {
	var unsupported []string
	found := false
	for line := range strings.SplitSeq(data, "\n") {
		line, _, _ = strings.Cut(line, "//")
		fields := strings.Fields(line)
		if len(fields) == 0 || (fields[0] != "go" && fields[0] != "toolchain") {
			continue
		}
		if fields[0] == "go" {
			if found {
				unsupported = append(unsupported, "duplicate go directive")
			}
			found = true
		}
		if len(fields) == 2 && fields[0] == "toolchain" && fields[1] == "default" {
			continue
		}
		if len(fields) != 2 {
			unsupported = append(unsupported, "go.mod toolchain directive is not qualified")
			continue
		}
		wanted := strings.TrimPrefix(fields[1], "go")
		v, ok := numbers(wanted)
		pin, _ := numbers(GoVersion)
		if !ok || v[0] != 1 || compare(v, pin) > 0 {
			unsupported = append(unsupported, "go.mod requires a Go toolchain beyond pinned "+GoVersion)
		}
	}
	if !found {
		unsupported = append(unsupported, "go.mod has no qualified go directive")
	}
	return unsupported
}

func compact(values []string) []string {
	out := values[:0]
	for _, v := range values {
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

// Nested projects are found by a bounded walk that never follows symlinks and
// skips vendored, generated and hidden trees. A nested go.mod provisions Go;
// one that cannot be read or qualified is reported with its path and never
// fails detection, so a stray file deep in a repository cannot block install.
// A nested Cargo.toml provisions Rust and a nested pubspec.yaml Flutter the
// same way. Other nested projects are
// only recorded as detected: Node and Python come with the image, and nested
// JVM builds (often an app's Android wrapper) are not provisioned and do not
// mark the environment degraded.
const maxNestedDepth = 3
const maxNestedEntries = 20000

var skippedDirs = map[string]bool{"node_modules": true, "vendor": true, "third_party": true, "testdata": true, "build": true, "dist": true, "target": true, "out": true}

func detectNested(root *os.Root, r *Requirements, found map[string]bool) {
	visited := 0
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		f, err := root.Open(dir)
		if err != nil {
			return
		}
		entries, err := f.ReadDir(maxRootEntries + 1)
		f.Close()
		if err != nil && err != io.EOF {
			return
		}
		for _, entry := range entries {
			if visited++; visited > maxNestedEntries {
				return
			}
			name := entry.Name()
			rel := name
			if dir != "." {
				rel = dir + "/" + name
			}
			if entry.IsDir() {
				if depth < maxNestedDepth && !strings.HasPrefix(name, ".") && !skippedDirs[name] {
					walk(rel, depth+1)
				}
				continue
			}
			if depth == 0 || !entry.Type().IsRegular() {
				continue
			}
			switch {
			case name == "go.mod":
				data, err := readManifest(root, rel)
				if err != nil {
					r.Unsupported = append(r.Unsupported, rel+" cannot be safely inspected")
					continue
				}
				found["go"] = true
				r.Go = true
				for _, problem := range goRequirements(string(data)) {
					r.Unsupported = append(r.Unsupported, rel+": "+problem)
				}
			case name == "pubspec.yaml":
				data, err := readManifest(root, rel)
				if err != nil {
					r.Unsupported = append(r.Unsupported, rel+" cannot be safely inspected")
					continue
				}
				found["flutter"] = true
				r.Flutter = true
				r.Unsupported = append(r.Unsupported, dartRequirements(dir+"/", string(data))...)
			case name == "package.json":
				found["node"] = true
			case name == "pyproject.toml" || (strings.HasPrefix(name, "requirements") && strings.HasSuffix(name, ".txt")):
				found["python"] = true
			case name == "Cargo.toml":
				data, err := readManifest(root, rel)
				if err != nil {
					r.Unsupported = append(r.Unsupported, rel+" cannot be safely inspected")
					continue
				}
				found["rust"] = true
				r.Rust = true
				r.Unsupported = append(r.Unsupported, rustRequirements(root, dir+"/", string(data))...)
			case name == "pom.xml" || strings.HasPrefix(name, "build.gradle"):
				found["jvm"] = true
			}
		}
	}
	walk(".", 0)
}

var rustVersionField = regexp.MustCompile(`^rust-version\s*=\s*"([^"]+)"`)
var toolchainChannel = regexp.MustCompile(`^channel\s*=\s*"([^"]+)"`)

// rustRequirements reports what the pinned Rust cannot satisfy for one crate
// or workspace: a newer rust-version, or a toolchain file pinning another
// release (without rustup, cargo would ignore it and build with the pinned
// one). Nothing is executed.
func rustRequirements(root *os.Root, dir, manifest string) []string {
	var unsupported []string
	pin, _ := numbers(RustVersion)
	for line := range strings.SplitSeq(manifest, "\n") {
		if m := rustVersionField.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			if v, ok := numbers(m[1]); !ok || compare(v, pin) > 0 {
				unsupported = append(unsupported, dir+"Cargo.toml requires Rust "+m[1]+" beyond pinned "+RustVersion)
			}
		}
	}
	for _, name := range []string{"rust-toolchain.toml", "rust-toolchain"} {
		data, err := readManifest(root, dir+name)
		if err != nil {
			continue
		}
		channel := strings.TrimSpace(string(data))
		for line := range strings.SplitSeq(string(data), "\n") {
			if m := toolchainChannel.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
				channel = m[1]
			}
		}
		if channel != "stable" && channel != RustVersion && channel != strings.Join(strings.Split(RustVersion, ".")[:2], ".") {
			unsupported = append(unsupported, dir+name+" pins a Rust toolchain other than the provisioned "+RustVersion)
		}
	}
	return unsupported
}

var dartSDKConstraint = regexp.MustCompile(`^\s+sdk:\s*['"]?([^'"#]+?)['"]?\s*(?:#.*)?$`)

// dartRequirements reports a pubspec whose Dart SDK constraint the pinned
// Flutter's Dart cannot satisfy. Only an `sdk:` line with a version
// constraint counts (dependencies' `sdk: flutter` is not one), and an
// unreadable constraint is left to `flutter pub get` rather than guessed.
func dartRequirements(dir, pubspec string) []string {
	for line := range strings.SplitSeq(pubspec, "\n") {
		m := dartSDKConstraint.FindStringSubmatch(line)
		if m == nil || strings.TrimSpace(m[1]) == "flutter" {
			continue
		}
		constraint := strings.TrimSpace(m[1])
		readable := true
		for _, part := range strings.Fields(operatorSpace.ReplaceAllString(constraint, "$1")) {
			if !selector.MatchString(part) {
				readable = false
			}
		}
		if readable && !matchesVersion(DartVersion, constraint) {
			return []string{dir + "pubspec.yaml requires Dart " + constraint + ", not the provisioned " + DartVersion}
		}
	}
	return nil
}
