package helm

import (
	"context"
	"fmt"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
