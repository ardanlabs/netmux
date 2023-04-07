package k8s

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ardanlabs.com/netmux/app/server/grpc"
	"github.com/ardanlabs.com/netmux/business/grpc/bridge"
	"github.com/sirupsen/logrus"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func resolveConfig(fname string) (*rest.Config, error) {

	if fname != "" {
		if fname == "~" {

			userHomeDir, err := os.UserHomeDir()
			if err != nil {
				fmt.Printf("error getting user home dir: %v\n", err)
				return nil, err
			}
			fname = filepath.Join(userHomeDir, ".kube", "config")
			fmt.Printf("Using kubeconfig: %s\n", fname)

		} else {
			if strings.HasPrefix(fname, "~") {
				userHomeDir, err := os.UserHomeDir()
				if err != nil {
					fmt.Printf("error getting user home dir: %v\n", err)
					return nil, err
				}
				fname = fname[2:]
				fname = filepath.Join(userHomeDir, fname)
			}
			fmt.Printf("Using kubeconfig: %s\n", fname)
		}
		return clientcmd.BuildConfigFromFlags("", fname)
	}
	logrus.Infof("Using InClusterConfig")
	return rest.InClusterConfig()
}

type Opts struct {
	Kubefile   string
	Namespaces []string
	All        bool
}

func MyNamespace() (string, error) {
	bs, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func runNamespace(ctx context.Context, cli *kubernetes.Clientset, ns string) error {

	logrus.Infof("K8s monigoring: %s", ns)

	handlePod := func(evt watch.EventType, dep *corev1.Pod) {

		if dep.Annotations["nx"] != "" {
			nxas, err := bridge.LoadBridges(dep.Annotations["nx"])
			if err != nil {
				logrus.Warnf("error reading annotation for %s.%s: %s", dep.Name, dep.Namespace, err.Error())
				return
			}
			for i := range nxas {
				nxa := nxas[i]
				if nxa.RemoteHost == "" {
					nxa.RemoteHost = dep.Status.PodIP
				}

				if nxa.RemotePort == "" {
					nxa.RemotePort = fmt.Sprintf("%v", dep.Spec.Containers[0].Ports[0].ContainerPort)
				}

				ep := bridge.ToProtoBufBridge(nxa)
				ep.K8Snamespace = dep.Namespace
				ep.K8Sname = dep.Name
				ep.K8Skind = dep.Kind

				switch evt {
				case watch.Added:
					grpc.Server().AddEp(ep)
				case watch.Deleted:
					grpc.Server().RemEp(ep)
				case watch.Modified:
					grpc.Server().AddEp(ep)
				}
			}
		}
	}

	handleService := func(evt watch.EventType, dep *corev1.Service) {
		if dep.Annotations["nx"] != "" {

			nxas, err := bridge.LoadBridges(dep.Annotations["nx"])
			if err != nil {
				logrus.Warnf("error reading annotation for %s.%s: %s", dep.Name, dep.Namespace, err.Error())
				return
			}
			for i := range nxas {
				nxa := nxas[i]
				if nxa.Name == "" {
					logrus.Warnf("Cant proceed w/o naming service: %s.%s", dep.Namespace, dep.Name)
					return
				}

				if nxa.RemoteHost == "" {
					nxa.RemoteHost = dep.Spec.ClusterIP
				}

				if nxa.RemotePort == "" {
					nxa.RemotePort = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
				}
				if nxa.LocalPort == "" {
					nxa.LocalPort = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
					nxa.LocalPort = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
					logrus.Warnf("Fixing w/o local port: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.Direction == "" {
					nxa.Direction = bridge.DirectionReward
				}

				ep := bridge.ToProtoBufBridge(nxa)
				ep.K8Snamespace = dep.Namespace
				ep.K8Sname = dep.Name
				ep.K8Skind = dep.Kind

				switch evt {
				case watch.Added:
					grpc.Server().AddEp(ep)
				case watch.Deleted:
					grpc.Server().RemEp(ep)
				case watch.Modified:
					grpc.Server().AddEp(ep)
				}
			}
		}
	}

	handleDeployments := func(evt watch.EventType, dep *appv1.Deployment) {
		if dep.Annotations["nx"] != "" {

			nxas, err := bridge.LoadBridges(dep.Annotations["nx"])
			if err != nil {
				logrus.Warnf("error reading annotation for %s.%s: %s", dep.Name, dep.Namespace, err.Error())
				return
			}

			for i := range nxas {
				nxa := nxas[i]

				if nxa.Name == "" {
					logrus.Warnf("Fixing Name: %s.%s", dep.Namespace, dep.Name)
					nxa.RemoteHost = dep.Name + "." + dep.Namespace

				}

				if nxa.RemoteHost == "" {
					nxa.RemoteHost = dep.Name + "." + dep.Namespace
					logrus.Warnf("Fixing remote addr: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.LocalHost == "" {
					nxa.LocalHost = dep.Name + "." + dep.Namespace
					logrus.Warnf("Fixing local addr: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.RemotePort == "" {
					nxa.RemotePort = fmt.Sprintf("%v", dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
					logrus.Warnf("Fixing w/o remote port: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.LocalPort == "" {
					nxa.LocalPort = fmt.Sprintf("%v", dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
					logrus.Warnf("Fixing w/o local port: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.Direction == "" {
					nxa.Direction = bridge.DirectionForward
				}

				ep := bridge.ToProtoBufBridge(nxa)
				ep.K8Snamespace = dep.Namespace
				ep.K8Sname = dep.Name
				ep.K8Skind = dep.Kind

				switch evt {
				case watch.Added:
					grpc.Server().AddEp(ep)
				case watch.Deleted:
					grpc.Server().RemEp(ep)
				case watch.Modified:
					grpc.Server().AddEp(ep)
				}
			}
		}
	}

	handleStatefulSets := func(evt watch.EventType, dep *appv1.StatefulSet) {
		if dep.Annotations["nx"] != "" {
			nxas, err := bridge.LoadBridges(dep.Annotations["nx"])
			if err != nil {
				logrus.Warnf("error reading annotation for %s.%s: %s", dep.Name, dep.Namespace, err.Error())
				return
			}

			for i := range nxas {
				nxa := nxas[i]

				if nxa.Name == "" {
					logrus.Warnf("Fixing Name: %s.%s", dep.Namespace, dep.Name)
					nxa.RemoteHost = dep.Name + "." + dep.Namespace

				}

				if nxa.RemoteHost == "" {
					nxa.RemoteHost = dep.Name + "." + dep.Namespace
					logrus.Warnf("Fixing remote addr: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.LocalHost == "" {
					nxa.LocalHost = dep.Name + "." + dep.Namespace
					logrus.Warnf("Fixing local addr: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.RemotePort == "" {
					nxa.RemotePort = fmt.Sprintf("%v", dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
					logrus.Warnf("Fixing w/o remote port: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.LocalPort == "" {
					nxa.LocalPort = fmt.Sprintf("%v", dep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
					logrus.Warnf("Fixing w/o local port: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.Direction == "" {
					nxa.Direction = bridge.DirectionForward
				}

				ep := bridge.ToProtoBufBridge(nxa)
				ep.K8Snamespace = dep.Namespace
				ep.K8Sname = dep.Name
				ep.K8Skind = dep.Kind

				switch evt {
				case watch.Added:
					grpc.Server().AddEp(ep)
				case watch.Deleted:
					grpc.Server().RemEp(ep)
				case watch.Modified:
					grpc.Server().AddEp(ep)
				}
			}
		}
	}

	logrus.Debugf("Loading resources")

	wpods, err := cli.CoreV1().Pods(ns).Watch(ctx, v1.ListOptions{})

	if err != nil {
		return err
	}

	wservices, err := cli.CoreV1().Services(ns).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	wdeployments, err := cli.AppsV1().Deployments(ns).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	wstatefulsets, err := cli.AppsV1().StatefulSets(ns).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case x := <-wpods.ResultChan():
				p, ok := x.Object.(*corev1.Pod)
				if ok && p != nil {
					handlePod(x.Type, x.Object.(*corev1.Pod))
				}

			case x := <-wservices.ResultChan():
				p, ok := x.Object.(*corev1.Service)
				if ok && p != nil {
					handleService(x.Type, p)
				}
			case x := <-wdeployments.ResultChan():
				p, ok := x.Object.(*appv1.Deployment)

				if ok && p != nil {
					handleDeployments(x.Type, p)
				}
			case x := <-wstatefulsets.ResultChan():
				p, ok := x.Object.(*appv1.StatefulSet)

				if ok && p != nil {
					handleStatefulSets(x.Type, p)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	logrus.Debugf("Namespace monitoring on")
	return nil

}

func Run(ctx context.Context, opts *Opts) error {
	if opts == nil {
		return fmt.Errorf("opts cant be nil")
	}

	kubeConfig, err := resolveConfig(opts.Kubefile)

	logrus.Infof("Running as: %s", kubeConfig.Username)

	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

	for _, v := range opts.Namespaces {
		err = runNamespace(ctx, clientset, v)
		if err != nil {
			return err
		}
	}

	<-ctx.Done()
	return nil
}
