package helm

import (
	"slices"
	"testing"
)

func TestVerbsFor(t *testing.T) {
	tests := []struct {
		name        string
		operation   Operation
		wantDesired []string
		wantCurrent []string
	}{
		{
			name:        "install",
			operation:   OperationInstall,
			wantDesired: []string{"get", "list", "watch", "create", "delete"},
		},
		{
			name:        "update",
			operation:   OperationUpdate,
			wantDesired: []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			wantCurrent: []string{"get", "list", "watch", "delete"},
		},
		{
			name:        "rollback",
			operation:   OperationRollback,
			wantDesired: []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			wantCurrent: []string{"get", "list", "watch", "delete"},
		},
		{
			name:        "uninstall",
			operation:   OperationUninstall,
			wantCurrent: []string{"get", "list", "watch", "delete"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired, current, err := verbsFor(tt.operation)
			if err != nil {
				t.Fatalf("verbsFor() error = %v", err)
			}
			if !slices.Equal(desired, tt.wantDesired) {
				t.Errorf("desired verbs = %v, want %v", desired, tt.wantDesired)
			}
			if !slices.Equal(current, tt.wantCurrent) {
				t.Errorf("current verbs = %v, want %v", current, tt.wantCurrent)
			}
		})
	}
}

func TestVerbsForUnknownOperation(t *testing.T) {
	desired, current, err := verbsFor(Operation("unknown"))
	if err == nil {
		t.Fatal("verbsFor() error = nil, want an error")
	}
	if desired != nil || current != nil {
		t.Fatalf("verbsFor() = (%v, %v), want nil verb lists", desired, current)
	}
}
