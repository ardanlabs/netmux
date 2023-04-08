package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/ardanlabs.com/netmux/business/grpc/clients/proxy"
	"github.com/ardanlabs.com/netmux/foundation/hosts"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type Context struct {
	parent *Netmux
	ctx    context.Context
	cancel func()

	Enabled           bool       `yaml:"enabled"`
	Name              string     `yaml:"name"`
	Services          []*Service `yaml:"services"`
	Url               string     `yaml:"url"`
	Key               string     `yaml:"key"`
	Runtime           Runtime    `yaml:"runtime"`
	EnablePortForward bool       `yaml:"enableportforward"`
	Auth              Auth       `yaml:"auth"`

	UrlAsUrl                  *url.URL    `yaml:"-"`
	PortForwardProcess        *os.Process `yaml:"-"`
	PortForwardStatus         Status
	uuidPortForwardTerminator string
	cli                       proxy.ProxyClient
	Token                     string
	NConns                    int
	Sent                      int64
	Recv                      int64
	chPfStop                  chan struct{}
	Status                    Status
}

func (c *Context) Cli() proxy.ProxyClient {
	return c.cli
}

func (c *Context) Prepare(cfg *Netmux) error {
	var err error
	c.parent = cfg
	c.UrlAsUrl, err = url.Parse(c.Url)
	if err != nil {
		return err
	}
	c.Runtime.Prepare(c)
	for _, v := range c.Services {
		v.Prepare(c)
	}
	c.PortForwardStatus = StatusStopped
	c.Status = StatusStopped
	return nil
}

func (c *Context) markContextDisabled() error {
	for _, s := range c.Services {
		s.Status = StatusDisabled
	}
	return nil

}

func (c *Context) ResetCounters() {
	for _, v := range c.Services {
		v.ResetCounters()
	}
	c.Sent = 0
	c.Recv = 0
}

func (c *Context) IsPortForwardRunning() bool {
	return c.chPfStop != nil
}

func (c *Context) StartPortForwarding() error {
	if c.chPfStop == nil {
		portInt, _ := strconv.Atoi(c.UrlAsUrl.Port())
		ch, err := PFStart(c.Runtime.Kubeconfig, c.Runtime.Kubecontext, &PortForwardAPodRequest{
			Pod: corev1.Pod{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      c.Runtime.Kubeservice,
					Namespace: c.Runtime.Kubenamespace,
				},
			},
			LocalPort: portInt,
			PodPort:   48080,
			Streams:   genericclioptions.IOStreams{},
		})
		if err != nil {
			return err
		}
		c.chPfStop = ch
	} else {
		logrus.Debugf("pf already running")
	}
	return nil

}

func (c *Context) StopPortForwarding() error {
	if c.chPfStop != nil {
		close(c.chPfStop)
		c.chPfStop = nil
		logrus.Debugf("Stopping port forward")
	} else {
		return fmt.Errorf("pf not running")
	}
	return nil
}

func (c *Context) onBridgeChange(ctx context.Context, bs *proxy.Bridges) error {
	for _, e := range bs.Eps {
		e := e
		switch e.Bridgeop {
		case "D":
			svc := c.SvcByName(e.Name)
			if svc != nil {
				svc.Stop()
				c.DelSvcByName(e.Name)
			}

		default:
			svc := c.SvcByName(e.Name)
			if svc != nil {
				svc.Stop()
				c.DelSvcByName(e.Name)
			}

			svc = &Service{
				parent: c,
				Bridge: bridge.New(e),
				hosts:  hosts.New(),
			}

			svc.ctx, svc.cancel = context.WithCancel(c.ctx)
			if svc.Bridge.Auto {
				err := svc.Start()
				if err != nil {
					return err
				}
			}
			c.Services = append(c.Services, svc)

		}

	}

	return nil
}

func (c *Context) Start(ctx context.Context, chReady chan struct{}) error {

	if c.Status != StatusStopped {
		return fmt.Errorf("ctx is not stopped")
	}
	var err error
	c.Status = StatusStarting
	defer func() {
		c.Status = StatusStopped

	}()

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.Status = StatusStarting
	logrus.Debugf("Loading ctx: %s", c.Name)

	if c.EnablePortForward {
		if c.UrlAsUrl.Hostname() == "localhost" {
			logrus.Debugf("Loading port forward for: %s -> %s", c.Name, c.Url)
			err := c.StartPortForwarding()
			if err != nil {
				return err
			}
			c.uuidPortForwardTerminator = TermHanlder.add(func() error {
				c.uuidPortForwardTerminator = ""
				return c.StopPortForwarding()
			})

			logrus.Debugf("Port forward loaded for: %s -> %s", c.Name, c.Url)

		}

	}

	c.cli, err = proxy.NewClient(c.UrlAsUrl.Host, c.Token)
	if err != nil {
		return err
	}

	//bridges, err := c.cli.GetConfigs(c.ctx, &proxy.Noop{})

	if err != nil {
		return err
	}

	//c.onBridgeChange(c.ctx, bridges)

	c.Status = StatusStarted

	go c.monitorBridges()

	chReady <- struct{}{}

	return nil
}
func (c *Context) monitorBridges() error {

	stream, err := c.cli.StreamConfig(c.ctx, &proxy.Noop{})
	if err != nil {
		return err
	}
	for {
		bs, err := stream.Recv()
		if err != nil {
			return err
		}
		err = c.onBridgeChange(c.ctx, bs)
		if err != nil {
			return err
		}
	}
}
func (c *Context) Stop() error {

	defer func() {
		c.Status = StatusStopped

	}()
	c.Status = StatusStopping
	if c.cancel != nil {
		c.cancel()
	}
	logrus.Infof("Stopping context: %s", c.Name)
	for _, s := range c.Services {
		err := s.Stop()
		if err != nil {
			logrus.Warnf("Error closing service %s: %s", s.Name(), err.Error())
		}

	}
	TermHanlder.TerminateSome(c.uuidPortForwardTerminator)
	return nil
}

func (c *Context) Login(user string, pass string) (string, error) {
	if c.EnablePortForward && !c.IsPortForwardRunning() {
		err := c.StartPortForwarding()
		if err != nil {
			return "", err
		}
		defer c.StopPortForwarding()
	}
	var err error
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.cli, err = proxy.NewClient(c.UrlAsUrl.Host, "")
	if err != nil {
		return "", err
	}
	res, err := c.cli.Login(c.ctx, &proxy.LoginReq{
		User: user,
		Pass: pass,
	})
	if err != nil {
		return "", err
	}

	c.Status = StatusStopped
	return res.Msg, nil
}

func (c *Context) Logout() error {
	if c.EnablePortForward && !c.IsPortForwardRunning() {
		err := c.StartPortForwarding()
		if err != nil {
			return err
		}
		defer c.StopPortForwarding()
	}
	var err error
	c.ctx, c.cancel = context.WithCancel(ctx)
	c.cli, err = proxy.NewClient(c.UrlAsUrl.Host, c.Token)
	if err != nil {
		return err
	}
	_, err = c.cli.Logout(c.ctx, &proxy.StringMsg{Msg: c.Token})
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) SvcByName(n string) *Service {
	for i := range c.Services {
		svc := c.Services[i]
		if svc.Name() == n {
			return svc
		}
	}
	return nil
}

func (c *Context) DelSvcByName(n string) {
	for i := range c.Services {
		svc := c.Services[i]
		if svc.Name() == n {
			c.Services[i] = c.Services[len(c.Services)-1]
			c.Services = c.Services[:len(c.Services)-1]
			return
		}
	}
}
