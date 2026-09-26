package helm

import (
	"context"
	"fmt"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
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
		return fmt.Errorf("check permission: %w", err)
	}

	if !review.Status.Allowed {
		return fmt.Errorf(
			"forbidden: cannot %s %s.%s in namespace %s: %s",
			verb,
			resource,
			group,
			namespace,
			review.Status.Reason,
		)
	}

	return nil
}

func (c *Client) CheckPermissions(
	ctx context.Context,
	operation Operation,
	manifest string,
) error {
	verbs, err := verbsFor(operation)
	if err != nil {
		return err
	}

	resources, err := c.ActionConfig.KubeClient.Build(
		strings.NewReader(manifest),
		false,
	)
	if err != nil {
		return fmt.Errorf("parse manifests: %w", err)
	}

	permissions := make(map[permission]struct{})

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

	for required := range permissions {
		if err := c.CanI(
			ctx,
			required.Verb,
			required.Group,
			required.Resource,
			required.Namespace,
		); err != nil {
			return err
		}
	}

	return nil
}

func verbsFor(operation Operation) ([]string, error) {
	switch operation {
	case OperationInstall:
		return []string{
			"create",
		}, nil

	case OperationUpdate:
		return []string{
			"get",
			"create",
			"update",
			"patch",
			"delete",
		}, nil

	case OperationUninstall:
		return []string{
			"get",
			"delete",
		}, nil

	default:
		return nil, fmt.Errorf(
			"unknown operation %q",
			operation,
		)
	}
}
