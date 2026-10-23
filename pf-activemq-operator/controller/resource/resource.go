// Package resource contains custom resource related functions
package resource

import (
	"context"
	"fmt"

	activemqartemisv1beta1 "github.com/arkmq-org/activemq-artemis-operator/v2/api/v1beta1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ExtraObjects identifies objects that should no longer exist. It ignores all objects that do not contain
// the label pf-activemq-operator.pnet.ch/primary-resource with the value matching the source object.
func ExtraObjects(ctx context.Context, cli client.Client, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ) ([]client.Object, error) {
	gvks := []schema.GroupVersionKind{
		{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    "Secret",
		},
		{
			Group:   v1.SchemeGroupVersion.Group,
			Version: v1.SchemeGroupVersion.Version,
			Kind:    "ConfigMap",
		},
		{
			Group:   activemqartemisv1beta1.GroupVersion.Group,
			Version: activemqartemisv1beta1.GroupVersion.Version,
			Kind:    "ActiveMQArtemis",
		},
		{
			Group:   certmanagerv1.SchemeGroupVersion.Group,
			Version: certmanagerv1.SchemeGroupVersion.Version,
			Kind:    "Certificate",
		},
	}

	result := []client.Object{}

	for i := range gvks {
		r, err := extraObjects(ctx, cli, pfCR, gvks[i])
		if err != nil {
			return nil, err
		}

		result = append(result, r...)
	}

	// fix, remove old secret to re-create keystore with new password
	// https://github.com/cert-manager/cert-manager/issues/3450
	certOutSecret := &v1.Secret{}

	err := cli.Get(ctx, client.ObjectKey{Name: fmt.Sprintf(DefaultAcceptorCertmgrOutSecretTpl, pfCR.Name), Namespace: pfCR.Spec.Instance.Namespace}, certOutSecret)
	if err == nil {
		result = append(result, certOutSecret)
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	return result, nil
}

func extraObjects(ctx context.Context, cli client.Client, pfCR *pfactivemqoperatorpnetchv1alpha1.PFActiveMQ, gvk schema.GroupVersionKind) ([]client.Object, error) {
	m := map[string]struct{}{}
	lo := &unstructured.UnstructuredList{}

	lo.SetGroupVersionKind(gvk)

	if err := cli.List(ctx, lo, client.InNamespace(pfCR.Spec.Instance.Namespace), client.MatchingLabels{
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNamespaceLabelKey: pfCR.Namespace,
		pfactivemqoperatorpnetchv1alpha1.PFActiveMQPrimaryResourceNameLabelKey:      pfCR.Name,
	}); err != nil {
		return nil, fmt.Errorf("failed to list object: %w", err)
	}

	result := make([]client.Object, 0, len(lo.Items))

	for i := range lo.Items {
		o := lo.Items[i]

		if _, ok := m[o.GetName()]; !ok {
			result = append(result, &o)
		}
	}

	return result, nil
}
