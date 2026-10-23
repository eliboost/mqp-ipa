// Package main The entry point for the controller
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log/slog"
	"os"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	activemqartemisv1beta1 "github.com/arkmq-org/activemq-artemis-operator/v2/api/v1beta1"
	pfactivemqoperatorpnetchv1alpha1 "gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/controller"

	// +kubebuilder:scaffold:imports
	"gitlab.pnet.ch/linux/go/slogcfg"
)

//nolint:gochecknoglobals // kubebuilder
var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

//nolint:gochecknoinits,wsl // kubebuilder
func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(pfactivemqoperatorpnetchv1alpha1.AddToScheme(scheme))

	utilruntime.Must(activemqartemisv1beta1.AddToScheme(scheme))

	// TODO: manuel??
	utilruntime.Must(certmanagerv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string

	var enableLeaderElection bool

	var probeAddr string

	var secureMetrics bool

	var enableHTTP2 bool

	var artemisImage string

	var artemisInitImage string

	var operationCommonName string

	var logLevel string

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metric endpoint binds to. "+
		"Use the port :8080. If not set, it will be 0 in order to disable the metrics server")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&artemisImage, "artemis-image", "middleware-docker.repo.pnet.ch/arkmq-org/activemq-artemis-broker-kubernetes:artemis.2.37.0", "The broker image for the artemis deployments")
	flag.StringVar(&artemisInitImage, "artemis-init-image", "middleware-docker-local.repo.pnet.ch/pf/middleware/artemis-custom-init-image:2.37.202410180858", "The init image for the artemis deployments")
	flag.StringVar(&operationCommonName, "operation-common-name", "a-pf-identity-mqp-ops-ui-e1", "The user with admin access on brokers with operations roles")
	flag.StringVar(&logLevel, "log-level", "info", "The level of logging: debug,info,warn,error")

	flag.Parse()

	level := slog.LevelInfo

	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		fmt.Printf("failed to read log level %s: %v", logLevel, err)

		os.Exit(1)
	}

	h := slogcfg.MustNewHandler(os.Stderr, slogcfg.WithFormat(slogcfg.FormatTypeJson), slogcfg.WithLevel(level))

	l := logr.FromSlogHandler(h)
	ctrl.SetLogger(l)

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")

		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	cfg := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "4dd45de6.pf-activemq-operator.pnet.ch",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	}

	ctx := ctrl.SetupSignalHandler()

	mgr, err := controller.New(ctx, ctrl.GetConfigOrDie(), cfg, artemisImage, artemisInitImage, operationCommonName)
	if err != nil {
		setupLog.Error(err, "failed to create manager")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
