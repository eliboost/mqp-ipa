// Package k8s capsules interaction logic with kubernetes
package k8s

import (
	"context"
)

// NamespaceClient defines interface for interacting with the k8s admission controller
type NamespaceClient interface {
	DryRunNamespace(ctx context.Context, namespace string, annotations map[string]string) error
}
