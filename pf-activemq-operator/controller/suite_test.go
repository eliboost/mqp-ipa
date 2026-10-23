package controller

import (
	"context"
	"fmt"
	"path/filepath"
	rtime "runtime"
	"testing"

	activemqartemisv1beta1 "github.com/arkmq-org/activemq-artemis-operator/v2/api/v1beta1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/stretchr/testify/require"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func setup(t *testing.T) (*setupResult, func() error) {
	ctx, cancel := context.WithCancel(context.Background())

	crdDirectoryPaths := []string{
		"../config/crd/bases",
	}

	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     crdDirectoryPaths,
		ErrorIfCRDPathMissing: true,
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "bin", "k8s",
			fmt.Sprintf("1.30.0-%s-%s", rtime.GOOS, rtime.GOARCH)),
	}

	teardown := func() error {
		cancel()
		return nil
	}

	cfg, err := testEnv.Start()
	require.NoError(t, err)
	require.NotNil(t, cfg)

	teardown = func() error {
		cancel()
		return testEnv.Stop()
	}

	admin, err := testEnv.AddUser(envtest.User{
		Name:   "myadmin",
		Groups: []string{"system:masters"},
	}, nil)
	if err != nil {
		t.Error(err)
	}

	if err := CreateOtherControllerResources(admin); err != nil {
		t.Error(err)
	}

	const clusterID = "e1-k8s-pfnet-lab-a"

	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = pfactivemqoperatorpnetchv1alpha1.AddToScheme(s)
	_ = activemqartemisv1beta1.AddToScheme(s)
	_ = certmanagerv1.AddToScheme(s)
	// +kubebuilder:scaffold:scheme

	k8sClient, err := client.New(admin.Config(), client.Options{Scheme: s})
	require.NoError(t, err)

	mgr, err := New(ctx, admin.Config(), ctrl.Options{
		Scheme:                 s,
		HealthProbeBindAddress: "0",
		LeaderElection:         false,
	}, "middleware-docker.repo.pnet.ch/arkmq-org/activemq-artemis-broker-kubernetes:artemis.2.37.0",
		"middleware-docker-local.repo.pnet.ch/pf/middleware/artemis-custom-init-image:2.37.202410180858", "test-operatorCommonName")
	if err != nil {
		t.Error(err)
	}

	go func() {
		err = mgr.Start(ctx)
		if err != nil {
			panic(err)
		}
	}()

	return &setupResult{
		k8sClient: k8sClient,
		clusterID: clusterID,
		admin:     admin,
	}, teardown
}

type setupResult struct {
	clusterID string
	k8sClient client.Client
	admin     *envtest.AuthenticatedUser
}
