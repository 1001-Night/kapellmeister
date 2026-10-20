package openstack

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gopkg.in/yaml.v3"
)

func TestCloudSelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clouds.yaml")
	if err := os.WriteFile(path, []byte("clouds:\n  only:\n    auth:\n      auth_url: https://example.invalid/v3\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OS_CLIENT_CONFIG_FILE", path)
	t.Setenv("OS_CLOUD", "")
	t.Setenv("OS_AUTH_URL", "")
	opts, err := clientOptions("", "", "")
	if err != nil || opts.Cloud != "only" {
		t.Fatalf("single default cloud: opts=%+v err=%v", opts, err)
	}
	t.Setenv("OS_AUTH_URL", "https://env.invalid/v3")
	opts, err = clientOptions("", "", "")
	if err != nil || opts.Cloud != "" {
		t.Fatalf("environment auth: opts=%+v err=%v", opts, err)
	}
	t.Setenv("OS_CLOUD", "from-env")
	opts, err = clientOptions("explicit", "region1", "internal")
	if err != nil || opts.Cloud != "explicit" || opts.RegionName != "region1" || opts.EndpointType != "internal" {
		t.Fatalf("explicit overrides: opts=%+v err=%v", opts, err)
	}
	opts, err = clientOptions("", "", "")
	if err != nil || opts.Cloud != "from-env" {
		t.Fatalf("OS_CLOUD: opts=%+v err=%v", opts, err)
	}
	t.Setenv("OS_CLOUD", "")
	t.Setenv("OS_AUTH_URL", "")
	if err := os.WriteFile(path, []byte("clouds:\n  a: {}\n  b: {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := clientOptions("", "", ""); err == nil {
		t.Fatal("multiple clouds must require selection")
	}
}

func TestKubeconfigWithMagnum(t *testing.T) {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test-ca"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	signs := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && r.URL.Path == "/clusters/demo":
			json.NewEncoder(w).Encode(map[string]string{"uuid": "cluster-uuid", "name": "demo", "api_address": "https://k8s.example:6443"})
		case r.Method == "GET" && r.URL.Path == "/certificates/cluster-uuid":
			json.NewEncoder(w).Encode(map[string]string{"pem": caPEM})
		case r.Method == "POST" && r.URL.Path == "/certificates":
			var body struct {
				Cluster string `json:"cluster_uuid"`
				CSR     string `json:"csr"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cluster != "cluster-uuid" {
				t.Errorf("invalid signing body: %+v %v", body, err)
				http.Error(w, "bad request", 400)
				return
			}
			block, _ := pem.Decode([]byte(body.CSR))
			if block == nil {
				t.Error("missing CSR")
				w.WriteHeader(400)
				return
			}
			csr, err := x509.ParseCertificateRequest(block.Bytes)
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if err := csr.CheckSignature(); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if csr.Subject.CommonName != "admin" || len(csr.Subject.Organization) != 1 || csr.Subject.Organization[0] != "system:masters" {
				t.Errorf("unexpected subject: %+v", csr.Subject)
			}
			certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
				SerialNumber: big.NewInt(2), Subject: csr.Subject,
				NotBefore: caCert.NotBefore, NotAfter: caCert.NotAfter,
				ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}, caCert, csr.PublicKey, caKey)
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			signs++
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"pem": string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := &gophercloud.ServiceClient{ProviderClient: &gophercloud.ProviderClient{HTTPClient: *server.Client()}, Endpoint: server.URL + "/"}
	data, err := Kubeconfig(context.Background(), client, "demo")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Current  string `yaml:"current-context"`
		Clusters []struct {
			Cluster map[string]string `yaml:"cluster"`
		} `yaml:"clusters"`
		Users []struct {
			User map[string]string `yaml:"user"`
		} `yaml:"users"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.Current != "demo" || len(config.Clusters) != 1 || len(config.Users) != 1 || signs != 1 {
		t.Fatal("invalid kubeconfig structure or signing request count")
	}
	decode := func(value string) []byte {
		t.Helper()
		b, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	if string(decode(config.Clusters[0].Cluster["certificate-authority-data"])) != caPEM || config.Clusters[0].Cluster["server"] != "https://k8s.example:6443" {
		t.Fatal("CA/server mismatch")
	}
	keyBlock, _ := pem.Decode(decode(config.Users[0].User["client-key-data"]))
	certBlock, _ := pem.Decode(decode(config.Users[0].User["client-certificate-data"]))
	if keyBlock == nil || certBlock == nil {
		t.Fatal("missing PEM")
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if !cert.PublicKey.(*rsa.PublicKey).Equal(&key.PublicKey) {
		t.Fatal("certificate and private key do not match")
	}
	if err := cert.CheckSignatureFrom(caCert); err != nil {
		t.Fatal(err)
	}
	if out, err := Kubeconfig(context.Background(), client, "missing"); err == nil || len(out) != 0 {
		t.Fatal("API errors must return no kubeconfig")
	}
}
