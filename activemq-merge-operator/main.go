// Package main provides an usage example
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/ardanlabs/conf/v3"
	"gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/internal/config"
	"gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/internal/instance"
	"gitlab.pnet.ch/middleware/activemq/activemq-merge-operator/internal/ldap"
	"gitlab.pnet.ch/middleware/activemq/pf-activemq-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/controller"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/render/markdown"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

//nolint:gochecknoglobals // will be filled by goreleaser
var (
	version = "0.0.0"
	commit  = "111111111111"
)

func main() {
	l := logr.FromSlogHandler(slog.NewTextHandler(os.Stderr, nil))
	ctrl.SetLogger(l)

	if err := run(l); err != nil {
		l.Error(err, "start failed")
		os.Exit(1)
	}
}

func run(l logr.Logger) error {
	// ━━ parse config  from env and config file ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	ctx := ctrl.SetupSignalHandler()

	c := config.Cfg{
		Build: config.Build{
			Version: version,
			Commit:  commit,
		},
	}

	const prefix = "MERGER"

	help, err := config.Parse(prefix, &c)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			fmt.Println(help)
			return nil
		}

		return err
	}

	if err := c.Validate(); err != nil {
		return fmt.Errorf("validation: %w", err)
	}

	mgr, err := runtime.New(ctx, l, c.RuntimeConfig())
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	appRoleGetter := func(mq v1alpha1.PFActiveMQ) ([]ldap.AppRole, error) {
		ownerRole := mq.Spec.Roles.Owner

		segments := strings.Split(ownerRole, ".")

		if len(segments) != 2 {
			return nil, fmt.Errorf("role %q contains not two segments after split by dot", ownerRole)
		}

		return []ldap.AppRole{
			{App: segments[0], Role: segments[1]},
		}, nil
	}

	ldapChain, err := ldap.ChangeChain(ldap.Config[v1alpha1.PFActiveMQ]{
		BaseURL:       c.LDAP.URL,
		Username:      c.LDAP.Username,
		Password:      c.LDAP.Password,
		AppRoleGetter: appRoleGetter,
	})
	if err != nil {
		return err
	}

	chain := policy.DefaultMergeRequestPolicyChain[v1alpha1.PFActiveMQ]().
		Append(
			policy.CheckFilePath[v1alpha1.PFActiveMQ](c.Filepath),
			policy.CheckIsMergingDeactivated[v1alpha1.PFActiveMQ](c.Deactivation),
		).
		Append(instance.ChangeChain(instance.Config{
			CesIdentityEndpoint: c.CES.CesIdentityEndpoint,
			ClusterInfoURL:      c.CES.ClusterInfoURL,
			ClusterInfoInsecure: c.CES.ClusterInfoInsecure,
		})...).
		Append(ldapChain...)

	const reviewerID = "reviewer"

	renderer, err := markdown.NewRenderer(c.Renderer)
	if err != nil {
		l.Error(err, "failed to create markdown renderer")
		os.Exit(1)
	}

	err = mgr.RegisterMergeRequestController(reviewerID, controller.Reviewer(controller.ReviewerConfig[v1alpha1.PFActiveMQ]{
		Renderer:                renderer,
		IgnoreMergeRequestChain: policy.DefaultMergeRequestIgnorePolicyChain(c.Gitlab.TargetBranch),
		ChangePolicyChain:       chain,
		Parse:                   instance.ParseFunc,
	}))
	if err != nil {
		l.Error(err, "failed to create reviewer")
		os.Exit(1)
	}

	err = mgr.RegisterMergeRequestController("merger", controller.Merger(controller.MergerConfig{
		IgnoreMergeRequestChain: policy.DefaultMergeRequestIgnorePolicyChain(c.Gitlab.TargetBranch).Append(policy.CheckAssignees),
		Reviewers:               []string{reviewerID},
	}))
	if err != nil {
		l.Error(err, "failed to register merger controller")
		os.Exit(1)
	}

	l.Info("start manager",
		"config", c,
	)

	err = mgr.Start(ctx)
	if err != nil {
		return fmt.Errorf("start manager: %w", err)
	}

	return nil
}
