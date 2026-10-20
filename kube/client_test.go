package kube

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"gitlab.com/1001n/kapellmeister/config"
)

func TestRESTClientGetterTimeout(t *testing.T) {
	getter, err := RESTClientGetter(config.Options{
		Timeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	flags := getter.(*genericclioptions.ConfigFlags)

	if flags.Timeout == nil {
		t.Fatal("Timeout = nil, want 30s")
	}
	if *flags.Timeout != "30s" {
		t.Fatalf("Timeout = %q, want %q", *flags.Timeout, "30s")
	}
}

func TestMemoryClientGetter(t *testing.T) {
	t.Setenv("KUBECONFIG", "must-not-load-local-config")
	raw := clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"first":  {Server: "https://first.example"},
			"second": {Server: "https://second.example"},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{"admin": {Token: "test-token"}},
		Contexts: map[string]*clientcmdapi.Context{
			"first":  {Cluster: "first", AuthInfo: "admin", Namespace: "original"},
			"second": {Cluster: "second", AuthInfo: "admin"},
		},
		CurrentContext: "first",
	}
	data, err := clientcmd.Write(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, contextName := range []string{"", "second"} {
		t.Run("context="+contextName, func(t *testing.T) {
			getter, err := RESTClientGetter(config.Options{
				KubeconfigData: data, Context: contextName, Namespace: "kapellmeister", Timeout: 30 * time.Second,
			})
			if err != nil {
				t.Fatal(err)
			}
			restConfig, err := getter.ToRESTConfig()
			if err != nil {
				t.Fatal(err)
			}
			wantHost := "https://first.example"
			if contextName == "second" {
				wantHost = "https://second.example"
			}
			if restConfig.Host != wantHost || restConfig.Timeout != 30*time.Second || restConfig.BearerToken != "test-token" {
				t.Fatalf("unexpected connection: host=%q timeout=%s", restConfig.Host, restConfig.Timeout)
			}
			namespace, _, err := getter.ToRawKubeConfigLoader().Namespace()
			if err != nil || namespace != "kapellmeister" {
				t.Fatalf("namespace = %q, error = %v", namespace, err)
			}
		})
	}
}

func TestMemoryClientGetterDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api":
			_, _ = w.Write([]byte(`{"kind":"APIVersions","apiVersion":"v1","versions":["v1"]}`))
		case "/apis":
			_, _ = w.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`))
		case "/api/v1":
			_, _ = w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"pods","kind":"Pod","namespaced":true,"verbs":["get","list"]}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	data, err := json.Marshal(map[string]any{
		"apiVersion": "v1", "kind": "Config", "current-context": "lab",
		"clusters": []any{map[string]any{"name": "lab", "cluster": map[string]string{"server": server.URL}}},
		"contexts": []any{map[string]any{"name": "lab", "context": map[string]string{"cluster": "lab"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	getter, err := RESTClientGetter(config.Options{KubeconfigData: data})
	if err != nil {
		t.Fatal(err)
	}
	mapper, err := getter.ToRESTMapper()
	if err != nil {
		t.Fatal(err)
	}
	mapping, err := mapper.RESTMapping(schema.GroupKind{Kind: "Pod"}, "v1")
	if err != nil {
		t.Fatal(err)
	}
	if mapping.Resource.Resource != "pods" || mapping.Scope.Name() != "namespace" {
		t.Fatalf("unexpected mapping: %s, scope=%s", mapping.Resource, mapping.Scope.Name())
	}
}

func TestMemoryClientGetterInvalidConfig(t *testing.T) {
	if _, err := RESTClientGetter(config.Options{KubeconfigData: []byte("clusters: [")}); err == nil {
		t.Fatal("malformed kubeconfig must return an error")
	}
}

func TestRESTClientGetterWithoutTimeout(t *testing.T) {
	getter, err := RESTClientGetter(config.Options{})
	if err != nil {
		t.Fatal(err)
	}
	flags := getter.(*genericclioptions.ConfigFlags)

	if flags.Timeout == nil {
		t.Fatal("Timeout = nil, want the client-go default")
	}
	if *flags.Timeout != "0" {
		t.Fatalf("Timeout = %q, want %q", *flags.Timeout, "0")
	}
}
