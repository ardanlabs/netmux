package service

import (
	"os"
	"runtime"
	"strings"
)

type Runtime struct {
	parent        *Context
	Arch          string      `yaml:"arch"`
	Type          RuntimeType `yaml:"type"`
	Kubeconfig    string      `yaml:"kubeconfig"`
	Kubecontext   string      `yaml:"kubecontext"`
	Kubenamespace string      `yaml:"kubenamespace"`
	Kubeinstance  string      `yaml:"kubeinstance"`
	Kubeservice   string      `yaml:"kubeservice"`
}

func (c *Runtime) Prepare(ctx *Context) {
	cfg := ctx.parent
	c.parent = ctx
	if c.Arch == "" {
		c.Arch = os.Getenv("GOARCH")
	}
	if c.Type == "" {
		c.Type = RuntimeKubernetes
	}

	if c.Kubenamespace == "" {
		c.Kubenamespace = "netmux"
	}

	if c.Kubeinstance == "" {
		c.Kubeinstance = "netmux"
	}

	if c.Arch == "" {
		c.Arch = runtime.GOARCH
	}

	if c.Kubeconfig == "KUBECONFIG" {
		if os.Getenv("KUBECONFIG") != "" {
			c.Kubeconfig = os.Getenv("KUBECONFIG")
		}
	}
	c.Kubeconfig = strings.Replace(c.Kubeconfig, "~", cfg.Userhome, 1)
	c.Kubecontext = strings.Replace(c.Kubecontext, "${USER}", cfg.Username, -1)
	c.Kubenamespace = strings.Replace(c.Kubenamespace, "${USER}", cfg.Username, -1)
	c.Kubeinstance = strings.Replace(c.Kubeinstance, "${USER}", cfg.Username, -1)
	c.Kubeservice = strings.Replace(c.Kubeservice, "${USER}", cfg.Username, -1)

	if c.Kubeservice == "" {
		c.Kubeservice = c.Kubeinstance
	}
}
