package controller

import (
	"context"
	"fmt"

	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/controller/resource"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const ingressDomainKey = "network.ingress.name"

// New creates new manager
func New(ctx context.Context, restCfg *rest.Config, cfg ctrl.Options, artemisImage, artemisInitImage, operationCommonName string) (manager.Manager, error) {
	mgr, err := ctrl.NewManager(restCfg, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller manager: %w", err)
	}

	ingressDomain, err := loadIngressDomain(ctx, restCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ingress domain: %w", err)
	}

	if err = (&PFActiveMQReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: Config{
			Artemis: resource.ArtemisConfig{
				IngressDomain: ingressDomain,
				Image:         artemisImage,
				InitImage:     artemisInitImage,
			},
			OperationCommonName: operationCommonName,
		},
	}).SetupWithManager(mgr); err != nil {
		return nil, fmt.Errorf("unable to create PFActiveMQ controller:%w", err)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up health check:%w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up ready check:%w", err)
	}

	return mgr, nil
}

func loadIngressDomain(ctx context.Context, cfg *rest.Config) (string, error) {
	clusterinfo := &v1.ConfigMap{}

	uncachedCli, err := client.New(cfg, client.Options{})
	if err != nil {
		return "", fmt.Errorf("failed to create uncached k8s uncachedCli:%w", err)
	}

	err = uncachedCli.Get(ctx, types.NamespacedName{Name: "clusterinfo", Namespace: "default"}, clusterinfo)
	if err != nil {
		return "", fmt.Errorf("unable to get clusterinfo:%w", err)
	}

	return clusterinfo.Data[ingressDomainKey], nil
}
