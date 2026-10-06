package helm

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"gitlab.com/1001n/kapellmeister/config"
)

func TestEnsureNamespaceCreatesOwnedNamespace(t *testing.T) {
	clientset := fake.NewClientset()
	client := &Client{
		KubeClient: clientset,
		Options: config.Options{
			Namespace: "kapellmeister",
		},
	}

	created, err := client.ensureNamespace(context.Background())
	if err != nil {
		t.Fatalf("ensureNamespace() error = %v", err)
	}
	if !created {
		t.Fatal("ensureNamespace() created = false, want true")
	}

	namespace, err := clientset.CoreV1().Namespaces().Get(
		context.Background(),
		"kapellmeister",
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("get namespace: %v", err)
	}
	if namespace.Labels[namespaceOwnerLabel] != "true" {
		t.Fatalf("owner label = %q, want true", namespace.Labels[namespaceOwnerLabel])
	}
}

func TestEnsureNamespaceKeepsExistingNamespace(t *testing.T) {
	clientset := fake.NewClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kapellmeister"},
	})
	client := &Client{
		KubeClient: clientset,
		Options: config.Options{
			Namespace: "kapellmeister",
		},
	}

	created, err := client.ensureNamespace(context.Background())
	if err != nil {
		t.Fatalf("ensureNamespace() error = %v", err)
	}
	if created {
		t.Fatal("ensureNamespace() created = true, want false")
	}

	namespace, err := clientset.CoreV1().Namespaces().Get(
		context.Background(),
		"kapellmeister",
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("get namespace: %v", err)
	}
	if _, exists := namespace.Labels[namespaceOwnerLabel]; exists {
		t.Fatal("existing namespace unexpectedly received an owner label")
	}
}

func TestDeleteOwnedNamespace(t *testing.T) {
	clientset := fake.NewClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kapellmeister",
			Labels: map[string]string{
				namespaceOwnerLabel: "true",
			},
		},
	})
	client := &Client{
		KubeClient: clientset,
		Options: config.Options{
			Namespace: "kapellmeister",
		},
	}

	if err := client.deleteOwnedNamespace(context.Background()); err != nil {
		t.Fatalf("deleteOwnedNamespace() error = %v", err)
	}

	_, err := clientset.CoreV1().Namespaces().Get(
		context.Background(),
		"kapellmeister",
		metav1.GetOptions{},
	)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("get namespace error = %v, want NotFound", err)
	}
}

func TestDeleteOwnedNamespaceKeepsUnownedNamespace(t *testing.T) {
	clientset := fake.NewClientset(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "kapellmeister"},
	})
	client := &Client{
		KubeClient: clientset,
		Options: config.Options{
			Namespace: "kapellmeister",
		},
	}

	if err := client.deleteOwnedNamespace(context.Background()); err != nil {
		t.Fatalf("deleteOwnedNamespace() error = %v", err)
	}

	_, err := clientset.CoreV1().Namespaces().Get(
		context.Background(),
		"kapellmeister",
		metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("existing namespace was deleted: %v", err)
	}
}

func TestAddNamespacePermissionForInstall(t *testing.T) {
	client := &Client{
		KubeClient: fake.NewClientset(),
		Options: config.Options{
			Namespace: "kapellmeister",
		},
	}
	permissions := make(map[permission]struct{})

	err := client.addNamespacePermission(
		context.Background(),
		permissions,
		OperationInstall,
	)
	if err != nil {
		t.Fatalf("addNamespacePermission() error = %v", err)
	}

	required := permission{Verb: "create", Resource: "namespaces"}
	if _, exists := permissions[required]; !exists {
		t.Fatalf("permissions do not contain %#v", required)
	}
}

func TestAddNamespacePermissionForUninstall(t *testing.T) {
	client := &Client{
		KubeClient: fake.NewClientset(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kapellmeister",
				Labels: map[string]string{
					namespaceOwnerLabel: "true",
				},
			},
		}),
		Options: config.Options{
			Namespace: "kapellmeister",
		},
	}
	permissions := make(map[permission]struct{})

	err := client.addNamespacePermission(
		context.Background(),
		permissions,
		OperationUninstall,
	)
	if err != nil {
		t.Fatalf("addNamespacePermission() error = %v", err)
	}

	required := permission{Verb: "delete", Resource: "namespaces"}
	if _, exists := permissions[required]; !exists {
		t.Fatalf("permissions do not contain %#v", required)
	}
}
