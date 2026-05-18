/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package markers_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"github.com/pt9912/k-deskflight/internal/audit/markers"
)

const sampleSource = `package sample

// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list
// +kubebuilder:rbac:groups=k-deskflight.geo-terrain.net,resources=opendeskpreflightchecks/status,verbs=update;patch

// SomeFunc is not relevant.
func SomeFunc() {}
`

func writeSample(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(path, []byte(sampleSource), 0o600); err != nil {
		t.Fatalf("write sample: %v", err)
	}
	return path
}

func TestParseRBACFileBasic(t *testing.T) {
	t.Parallel()
	path := writeSample(t)
	got, err := markers.ParseRBACFile(path)
	if err != nil {
		t.Fatalf("ParseRBACFile: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("count: got %d, want 3", len(got))
	}

	// Storage-Marker
	if !reflect.DeepEqual(got[0].Groups, []string{"storage.k8s.io"}) {
		t.Errorf("Groups[0]: got %+v", got[0].Groups)
	}
	if !reflect.DeepEqual(got[0].Resources, []string{"storageclasses"}) {
		t.Errorf("Resources[0]: got %+v", got[0].Resources)
	}
	if !reflect.DeepEqual(got[0].Verbs, []string{"get", "list", "watch"}) {
		t.Errorf("Verbs[0]: got %+v", got[0].Verbs)
	}

	// Core-Group ("")
	if !reflect.DeepEqual(got[1].Groups, []string{""}) {
		t.Errorf("Groups[1] (core): got %+v, want [\"\"]", got[1].Groups)
	}

	// Sub-Resource
	if got[2].Resources[0] != "opendeskpreflightchecks/status" {
		t.Errorf("Resources[2] (subresource): got %q", got[2].Resources[0])
	}
}

func TestMarkerExpand(t *testing.T) {
	t.Parallel()
	m := markers.Marker{
		Groups:    []string{"g1"},
		Resources: []string{"r1", "r2"},
		Verbs:     []string{"get", "list"},
	}
	got := m.Expand()
	if len(got) != 4 {
		t.Fatalf("Expand count: got %d, want 4", len(got))
	}
	gotStr := make([]string, 0, len(got))
	for _, tr := range got {
		gotStr = append(gotStr, tr.String())
	}
	sort.Strings(gotStr)
	want := []string{
		"get g1/r1",
		"get g1/r2",
		"list g1/r1",
		"list g1/r2",
	}
	if !reflect.DeepEqual(gotStr, want) {
		t.Errorf("Expand strings: got %+v, want %+v", gotStr, want)
	}
}

func TestTripleStringCoreGroup(t *testing.T) {
	t.Parallel()
	tr := markers.Triple{Group: "", Resource: "nodes", Verb: "list"}
	if got, want := tr.String(), "list core/nodes"; got != want {
		t.Errorf("String: got %q, want %q (Group=\"\" rendert als 'core')", got, want)
	}
}

func TestParseRBACFileNotFound(t *testing.T) {
	t.Parallel()
	if _, err := markers.ParseRBACFile("/nonexistent/path.go"); err == nil {
		t.Errorf("expected error for nonexistent file")
	}
}
