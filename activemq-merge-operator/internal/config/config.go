// Package config is the main entry point for the controller.
package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/ardanlabs/conf/v3"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/policy"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/render/markdown"
	"gitlab.pnet.ch/kubernetes/gitlab-controller-runtime/pkg/runtime"
	"gopkg.in/yaml.v3"
)

// Parse parses the specified config struct.
func Parse(prefix string, c *Cfg) (help string, err error) {
	if c == nil {
		return "", errors.New("nil config pointer")
	}

	help, err = conf.Parse(prefix, c)
	if err != nil {
		return help, err
	}

	d, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", c.ConfigFile, err)
	}

	if err := yaml.Unmarshal(d, &c); err != nil {
		return "", fmt.Errorf("unmarshal file %s: %w", c.ConfigFile, err)
	}

	return "", nil
}

// Cfg compose the configs to load from env and yaml
type Cfg struct {
	Build        Build
	ConfigFile   string                    `conf:"help:The path to the configuration file,required,env:CONFIG"`
	Filepath     *regexp.Regexp            `yaml:"file-path,omitempty"`
	Deactivation policy.DeactivationConfig `yaml:"deactivation"`
	Renderer     markdown.Config           `yaml:"renderer"`
	Gitlab       struct {
		BaseURL      string `conf:"default:https://gitlab.pnet.ch/api/v4,help:The gitlab server URL"`
		Token        string `conf:"help:The gitlab server token,required"`
		Repository   string `conf:"help:The gitlab repository where the operator merges merge requests,required"`
		TargetBranch string `conf:"help:Ignore all merge with other target branches,default:main"`
	}
	NATS struct {
		Servers       []string `conf:"help:A list of nats servers to connect to,default:p1-linux-mlsu014.pnet.ch:7001;p1-linux-mlsu015.pnet.ch:7001;p1-linux-mlsu016.pnet.ch:7001"`
		Seed          string   `conf:"help:NKey seed to for authentication"`
		SubjectPrefix string   `conf:"help:The subject the operator subscribes to,default:gitlab"`
		LogPayload    bool     `conf:"help:Log the message payload in the generic event source"`
	}
	Manager struct {
		MetricsAddr    string        `conf:"default::8080,help:The address the metric endpoint binds to"`
		ProbeAddr      string        `conf:"default::8081,help:The address the probe endpoint binds to"`
		ResyncInterval time.Duration `conf:"default:900s,help:The interval at which the sever resyncs all merge requests."`
		LeaderElection bool          `conf:"help:Start the manager with a leader election"`
	}
	Outproxy struct {
		URL         string        `conf:"help:The URL the the outproxy grpc endpoint,default:https://outproxy.p1-management-a.pnet.ch:7002"`
		DialTimeout time.Duration `conf:"help:The maximum amount of time to wait for the connection to be established,default:5s"`
	}
	LDAP struct {
		URL      string `conf:"help:The BaseURL for pfdir rest api,default:https://pfdir2.pnet.ch:2636"`
		Username string `conf:"help:The Username for pfdir rest api"`
		Password string `conf:"help:The Password for pfdir rest api"`
	}
	CES struct {
		ClusterInfoURL      string `conf:"help:The url for the clusterinfo endpoint,default:https://clusterinfo.p1-management-a.pnet.ch"`
		ClusterInfoInsecure bool   `conf:"help:Disable tls verification,default:false"`
		CesIdentityEndpoint string `conf:"The url for the ces identity endpoint,default:https://p1-ces-http-a.pnet.ch/ces/api/public/v1/mqp/users/"`
	}
}

// Validate validates controller config
func (c Cfg) Validate() (err error) {
	if err := c.RuntimeConfig().Validate(); err != nil {
		return fmt.Errorf("runtime validation: %w", err)
	}

	return nil
}

// Build represents the current version (with git commit revision).
type Build struct {
	Version string
	Commit  string
}

// RuntimeConfig create gitlab runtime configuration.
func (c Cfg) RuntimeConfig() runtime.Config {
	return runtime.Config{
		Manager: runtime.ManagerConfig{
			MetricsAddr:    c.Manager.MetricsAddr,
			ProbeAddr:      c.Manager.ProbeAddr,
			ResyncInterval: c.Manager.ResyncInterval,
			LeaderElection: c.Manager.LeaderElection,
		},
		Nats: runtime.NatsConfig{
			Servers:       c.NATS.Servers,
			Seed:          c.NATS.Seed,
			SubjectPrefix: c.NATS.SubjectPrefix,
			LogPayload:    c.NATS.LogPayload,
		},
		Gitlab: runtime.Gitlab{
			BaseURL:      c.Gitlab.BaseURL,
			Repository:   c.Gitlab.Repository,
			Token:        c.Gitlab.Token,
			TargetBranch: c.Gitlab.TargetBranch,
		},
	}
}
