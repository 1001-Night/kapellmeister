package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadValuesWithoutFile(t *testing.T) {
	values, err := loadValues("")
	if err != nil {
		t.Fatalf("loadValues() error = %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("loadValues() = %v, want empty values", values)
	}
}

func TestLoadValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.yaml")
	data := []byte("replicaCount: 3\nimage:\n  tag: 1.2.3\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write values file: %v", err)
	}

	values, err := loadValues(path)
	if err != nil {
		t.Fatalf("loadValues() error = %v", err)
	}

	want := map[string]interface{}{
		"replicaCount": float64(3),
		"image": map[string]interface{}{
			"tag": "1.2.3",
		},
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("loadValues() = %#v, want %#v", values, want)
	}
}

func TestLoadValuesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.yaml")

	_, err := loadValues(path)
	if err == nil {
		t.Fatal("loadValues() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("loadValues() error = %q, want file path", err)
	}
}

func TestLoadValuesInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "values.yaml")
	if err := os.WriteFile(path, []byte("replicaCount: [\n"), 0o600); err != nil {
		t.Fatalf("write values file: %v", err)
	}

	_, err := loadValues(path)
	if err == nil {
		t.Fatal("loadValues() error = nil, want a YAML parsing error")
	}
}
