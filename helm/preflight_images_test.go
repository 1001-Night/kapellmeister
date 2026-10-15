package helm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestImageRegistries(t *testing.T) {
	manifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: registry.example.com:5000/tools/init:1
      containers:
        - name: app
          image: nginx:1.27
        - name: sidecar
          image: docker.io/library/busybox:1
---
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: cleanup
              image: registry.example.com:5000/tools/cleanup:1
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ignored
data:
  image: unreachable.example.com/test
`
	got, err := imageRegistries(manifest)
	if err != nil {
		t.Fatalf("imageRegistries() error = %v", err)
	}
	want := []string{"registry-1.docker.io", "registry.example.com:5000"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("imageRegistries() = %v, want %v", got, want)
	}
}

func TestPreflightImageRegistries(t *testing.T) {
	for _, test := range []struct {
		name       string
		statusCode int
		wantError  bool
	}{
		{name: "available", statusCode: http.StatusOK},
		{name: "authentication required", statusCode: http.StatusUnauthorized},
		{name: "access denied", statusCode: http.StatusForbidden, wantError: true},
		{name: "unavailable", statusCode: http.StatusServiceUnavailable, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.URL.Path != "/v2/" {
					t.Errorf("request path = %q, want /v2/", r.URL.Path)
				}
				w.WriteHeader(test.statusCode)
			}))
			defer server.Close()

			image := strings.TrimPrefix(server.URL, "http://") + "/app:1"
			manifest := fmt.Sprintf("apiVersion: v1\nkind: Pod\nmetadata:\n  name: app\nspec:\n  containers:\n    - name: app\n      image: %s\n", image)
			err := preflightImageRegistries(context.Background(), manifest)
			if (err != nil) != test.wantError {
				t.Fatalf("preflightImageRegistries() error = %v, wantError %v", err, test.wantError)
			}
			if calls != 1 {
				t.Errorf("requests = %d, want 1", calls)
			}
		})
	}
}

func TestImageRegistriesRejectsEmptyImage(t *testing.T) {
	manifest := "apiVersion: v1\nkind: Pod\nmetadata:\n  name: app\nspec:\n  containers:\n    - name: app\n      image: ''\n"
	if _, err := imageRegistries(manifest); err == nil {
		t.Fatal("imageRegistries() error = nil, want empty image error")
	}
}
