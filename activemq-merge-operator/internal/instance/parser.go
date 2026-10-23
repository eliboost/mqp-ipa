// Package instance showcase the implementation of a policy module
package instance

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"

	activemqv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
)

// ParseFunc used to parse string data to activemqv1alpha1.PFActiveMQ
func ParseFunc(filePath, data string) (*activemqv1alpha1.PFActiveMQ, error) {
	return makeInstance(filePath, []byte(data))
}

func makeInstance(_ string, data []byte) (*activemqv1alpha1.PFActiveMQ, error) {
	amq := &activemqv1alpha1.PFActiveMQ{}

	sch := runtime.NewScheme()
	_ = scheme.AddToScheme(sch)
	_ = activemqv1alpha1.AddToScheme(sch)

	_, _, err := serializer.NewCodecFactory(sch).UniversalDeserializer().Decode(data, nil, amq)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing activemq object")
	}

	return amq, nil
}
