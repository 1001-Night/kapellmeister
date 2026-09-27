package helm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type permission struct {
	Verb      string
	Group     string
	Resource  string
	Namespace string
}

type Operation string

const (
	OperationInstall   Operation = "install"
	OperationUpdate    Operation = "update"
	OperationUninstall Operation = "uninstall"
)

func (c *Client) CanI(
	ctx context.Context,
	verb string,
	group string,
	resource string,
	namespace string,
) error {
	review, err := c.KubeClient.
		AuthorizationV1().
		SelfSubjectAccessReviews().
		Create(
			ctx,
			&authorizationv1.SelfSubjectAccessReview{
				Spec: authorizationv1.SelfSubjectAccessReviewSpec{
					ResourceAttributes: &authorizationv1.ResourceAttributes{
						Verb:      verb,
						Group:     group,
						Resource:  resource,
						Namespace: namespace,
					},
				},
			},
			metav1.CreateOptions{},
		)
	if err != nil {
		return fmt.Errorf("check permission %s %s: %w", verb, resource, err)
	}

	if review.Status.Allowed {
		return nil
	}

	apiResource := resource
	if group != "" {
		apiResource += "." + group
	}

	return fmt.Errorf(
		"cannot %s %s in namespace %q: %s",
		verb,
		apiResource,
		namespace,
		review.Status.Reason,
	)
}

func (c *Client) CheckPermissions(
	ctx context.Context,
	operation Operation,
	desiredManifest string,
	currentManifest string,
) error {
	// проверка прав для двух манифестов (два из-за update)
	desiredVerbs, currentVerbs, err := verbsFor(operation)
	if err != nil {
		return err
	}

	permissions := make(map[permission]struct{})

	if err := c.addManifestPermissions(permissions, desiredManifest, desiredVerbs); err != nil {
		return err
	}
	if err := c.addManifestPermissions(permissions, currentManifest, currentVerbs); err != nil {
		return err
	}

	c.addHelmStoragePermissions(permissions, operation)

	if operation == OperationInstall {
		if err := c.addNamespacePermission(ctx, permissions); err != nil {
			return err
		}
	}

	return c.checkAllPermissions(ctx, permissions)
}

func (c *Client) addManifestPermissions(
	permissions map[permission]struct{},
	manifest string,
	verbs []string,
) error {
	if strings.TrimSpace(manifest) == "" {
		return nil
	}

	resources, err := c.ActionConfig.KubeClient.Build(
		strings.NewReader(manifest),
		false,
	)
	if err != nil {
		return fmt.Errorf("parse manifests: %w", err)
	}

	for _, resource := range resources {
		namespace := resource.Namespace
		if resource.Mapping.Scope.Name() == meta.RESTScopeNameRoot {
			namespace = ""
		}

		for _, verb := range verbs {
			permissions[permission{
				Verb:      verb,
				Group:     resource.Mapping.Resource.Group,
				Resource:  resource.Mapping.Resource.Resource,
				Namespace: namespace,
			}] = struct{}{}
		}
	}

	return nil
}

func (c *Client) addHelmStoragePermissions(
	permissions map[permission]struct{},
	operation Operation,
) {
	// права на secrets
	var verbs []string

	switch operation {
	case OperationInstall:
		verbs = []string{"get", "list", "create", "update", "delete"}
	case OperationUpdate:
		verbs = []string{"get", "list", "create", "update", "delete"}
	case OperationUninstall:
		verbs = []string{"get", "list", "update", "delete"}
	}

	for _, verb := range verbs {
		permissions[permission{
			Verb:      verb,
			Group:     "",
			Resource:  "secrets",
			Namespace: c.Options.Namespace,
		}] = struct{}{}
	}
}

func (c *Client) addNamespacePermission(
	ctx context.Context,
	permissions map[permission]struct{},
) error {
	_, err := c.KubeClient.
		CoreV1().
		Namespaces().
		Get(ctx, c.Options.Namespace, metav1.GetOptions{})

	switch {
	case err == nil:
		return nil
	case apierrors.IsNotFound(err):
		permissions[permission{
			Verb:      "create",
			Group:     "",
			Resource:  "namespaces",
			Namespace: "",
		}] = struct{}{}
		return nil
	default:
		return fmt.Errorf("check namespace %q: %w", c.Options.Namespace, err)
	}
}

func (c *Client) checkAllPermissions(
	ctx context.Context,
	permissions map[permission]struct{},
) error {
	var denied []string

	// все отказы в правах
	for required := range permissions {
		if err := c.CanI(
			ctx,
			required.Verb,
			required.Group,
			required.Resource,
			required.Namespace,
		); err != nil {
			denied = append(denied, err.Error())
		}
	}

	if len(denied) == 0 {
		return nil
	}

	sort.Strings(denied)
	return fmt.Errorf(
		"missing permissions:\n- %s",
		strings.Join(denied, "\n- "),
	)
}

func verbsFor(
	operation Operation,
) (
	desired []string,
	current []string,
	err error,
) {
	switch operation {
	case OperationInstall:
		return []string{"get", "list", "watch", "create", "delete"}, nil, nil
	case OperationUpdate:
		return []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			[]string{"get", "list", "watch", "delete"},
			nil
	case OperationUninstall:
		return nil,
			[]string{"get", "list", "watch", "delete"},
			nil
	default:
		return nil, nil, fmt.Errorf("unknown operation %q", operation)
	}
}
