// Package gitlab capsules interaction logic with gitlab
package gitlab

// NamespaceClient defines interface for getting namespace data
type NamespaceClient interface {
	GetNamespaceConfig(cluster string, namespace string) (project map[string]any, retry bool, err error)
}
