package commands

type Opts struct {
	User    string `short:"u"`
	Pass    string `short:"p"`
	CmdLine string `kong:"-" json:"cmd_line,omitempty"`
	//Ctxs    []string `default:"*" name:"ctx" help:"Contexts to Load" json:"ctx,omitempty"`

	Exit struct {
	} `cmd:"" help:"Exits daemon"`

	Load struct {
		Fname string `help:"file to load, defaults to ~/.netmux.yaml"`
	} `cmd:"" help:"Loads config"`

	Webview struct {
	} `cmd:"" help:"Shows GUI Webview"`

	Tray struct {
		Install   struct{} `cmd:"" help:"Tray notification"`
		Uninstall struct{} `cmd:"" help:"Tray notification"`
		Run       struct{} `cmd:"" help:"Tray notification"`
		Disable   struct{} `cmd:"" help:"Tray notification"`
		Enable    struct{} `cmd:"" help:"Tray notification"`
	} `cmd:"" help:"Tray notification"`
	Status struct {
		Zero   bool `short:"z" default:"false" help:"Zero counters"`
		Repeat bool `short:"r" default:"false" help:"Keeps repeating"`
	} `cmd:"" help:"Prints status on the screen"`

	Ctx struct {
		Install struct {
			Ctx  string `arg:"" help:"Context name"`
			Kctx string `arg:"" help:"Context name"`
			Ns   string `arg:"" short:"s" help:"Namespace to deploy to - *"`
			Arch string `arg:"" help:"Architecture: arm64 or amd64"`
			//Dumponly bool   `short:"d" help:"Only dumps yaml manifest"`
		} `cmd:"" help:"Installs agent into cluster"`

		Uninstall struct {
			Ctx       string `arg:"" help:"Context name"`
			Namespace string `short:"s" help:"Namespace to deploy to - *"`
		} `cmd:"" help:"Uninstall from cluster"`

		Login struct {
			Ctx string `arg:"" help:"Context name"`
			On  bool   `help:"If set, will trigger context on after login"`
		} `cmd:"" help:"Logon into context"`

		Logout struct {
			Ctx string `arg:"" help:"Context name"`
		} `cmd:"" help:"Logout from context"`

		On struct {
			Ctx    string `arg:"" help:"Context name"`
			Noauto bool   `short:"n" help:"No auto connections are established"`
		} `cmd:"" help:"Turn context on"`

		Off struct {
			Ctx string `arg:"" help:"Context name"`
		} `cmd:"" help:"Turn context off"`
		Pfon struct {
			Ctx string `arg:"" help:"Context name"`
		} `cmd:"" help:"Turn context port forward on"`
		Pfoff struct {
			Ctx string `arg:"" help:"Context name"`
		} `cmd:"" help:"Turn context port forward off"`
		Reset struct {
			Ctx string `arg:"" help:"Context name"`
		} `cmd:"" help:"Resets a context"`
		Ping struct {
			Ctx string   `arg:"" help:"Context name"`
			Cmd []string `arg:"" help:"cmd complementary info"`
		} `cmd:"" help:"Ping a host from cluster proxy"`
		Pscan struct {
			Ctx string   `arg:"" help:"Context name"`
			Cmd []string `arg:"" help:"cmd complementary info"`
		} `cmd:"" help:"Runs portscan from cluster proxy"`
		Nc struct {
			Ctx string   `arg:"" help:"Context name"`
			Cmd []string `arg:"" help:"cmd complementary info"`
		} `cmd:"" help:"Runs Netcat (nc) from cluster proxy"`
		Speedtest struct {
			Ctx string `arg:"" help:"Context name"`
			Pl  string `arg:"" help:"Payload size"`
		} `cmd:"" help:"Tests download speed from this context"`
	} `cmd:"" help:"Netmux contexts related commands"`

	Svc struct {
		On struct {
			Ctx string   `arg:"" help:"Context name"`
			Svc []string `arg:"" help:"Service name"`
		} `cmd:"" help:"Turns service on"`

		Off struct {
			Ctx string   `arg:"" help:"Context name"`
			Svc []string `arg:"" help:"Service name"`
		} `cmd:"" help:"Turns service off"`
		Reset struct {
			Ctx string   `arg:"" help:"Context name"`
			Svc []string `arg:"" help:"Service name"`
		} `cmd:"" help:"Resets a service"`
	} `cmd:"" help:"Netmux services related commands"`

	Auth struct {
		Hash struct {
		} `cmd:"" help:"Gens hash from pass (-p), to be used in passwd files"`
	} `cmd:"" help:"Authentication related commands"`

	Config struct {
		Set struct {
			Fname string `arg:"" help:"File name to be used"`
		} `cmd:"" help:"Sets config file to be used"`
		Hosts struct {
			Show  struct{} `cmd:"" help:"Shows hosts file"`
			Reset struct{} `cmd:"" help:"Resets hosts file"`
		} `cmd:"" help:"Dns entries related commands"`
		Get  struct{} `cmd:"" help:"Tells what is the config being used" `
		Dump struct{} `cmd:"" help:"Dumps config and exit" `
	} `cmd:"" help:"Agent configuration related commands"`

	Logs struct {
	} `cmd:"" help:"Follows agent logs"`

	Server struct {
	} `cmd:"" help:"Runs agent as a server"`

	Monitor struct {
	} `cmd:"" help:"Text UI for monitoring agent" `

	Local struct {
		Autoinstall struct {
			Ns      string `default:"default" help:"Defines the namespace to be used - if * is provided, global deploy will be used (ClusterRoles will apply - hope you have the rights)."`
			Ctx     string `default:"default" help:"Context to be used from existing kube config"`
			Arch    string `default:"" help:"Architecture: arm64 or amd64 - defaults to your machine arch"`
			Autopub bool   `default:"false" help:"If set, netmux server will autopublish pods and services"`
		} `cmd:"" help:"installs the agent and sets up the end in a single shot. (requires root)"`
		Install struct {
			Ns  string `default:"default" help:"Defines the namespace to be used - if * is provided, global deploy will be used (ClusterRoles will apply - hope you have the rights)."`
			Ctx string `default:"default" help:"Context to be used from existing kube config"`
		} `cmd:"" help:"installs agent (requires root)"`
		Uninstall struct {
		} `cmd:"" help:"uninstalls agent (requires root)"`
	} `cmd:"" help:"Agent related commands"`
}

//func (c *Opts) HasCtx(ctx string) bool {
//	for _, v := range c.Ctxs {
//		if v == ctx {
//			return true
//		}
//	}
//	return false
//}

var opts Opts
