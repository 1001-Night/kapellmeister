package kube

import (
	"testing"
	"time"

	"gitlab.com/1001n/kapellmeister/config"
)

func TestRESTClientGetterTimeout(t *testing.T) {
	getter := RESTClientGetter(config.Options{
		Timeout: 30 * time.Second,
	})

	if getter.Timeout == nil {
		t.Fatal("Timeout = nil, want 30s")
	}
	if *getter.Timeout != "30s" {
		t.Fatalf("Timeout = %q, want %q", *getter.Timeout, "30s")
	}
}

func TestRESTClientGetterWithoutTimeout(t *testing.T) {
	getter := RESTClientGetter(config.Options{})

	if getter.Timeout == nil {
		t.Fatal("Timeout = nil, want the client-go default")
	}
	if *getter.Timeout != "0" {
		t.Fatalf("Timeout = %q, want %q", *getter.Timeout, "0")
	}
}
