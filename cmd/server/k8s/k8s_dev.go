package k8s

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/cmd/server/grpc"
	pb "go.digitalcircle.com.br/dc/netmux/lib/proto/server"
	"go.digitalcircle.com.br/dc/netmux/lib/types"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

func RunDev(ctx context.Context, opts *Opts) error {
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

	if len(opts.Namespaces) != 1 {
		return fmt.Errorf("must provide exactly 1 namespace for DevSimpleNS mode")
	}

	err = runNamespaceDevSimpleNS(ctx, clientset, opts.Namespaces[0], opts.All)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func runNamespaceDevSimpleNS(ctx context.Context, cli *kubernetes.Clientset, ns string, all bool) error {

	logrus.Infof("K8s monigoring: %s", ns)

	handleService := func(evt watch.EventType, dep *corev1.Service) {
		if dep.Annotations["nx"] != "" {

			nxas := types.NewBridges()
			err := nxas.LoadFromAnnotation(dep.Annotations["nx"])

			if err != nil {
				logrus.Warnf("error reading annotation for %s.%s: %s", dep.Name, dep.Namespace, err.Error())
				return
			}
			for i := range nxas {
				nxa := nxas[i]
				if nxa.Name == "" {
					nxa.Name = dep.Name
					logrus.Warnf("Using name from service: %s.%s", dep.Namespace, dep.Name)
				}

				if nxa.RemoteAddr == "" {
					nxa.RemoteAddr = dep.Spec.ClusterIP
				}

				if nxa.LocalAddr == "" {
					nxa.LocalAddr = dep.Name
				}

				if nxa.RemotePort == "" {
					nxa.RemotePort = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
					logrus.Warnf("Fixing w/o remote port: %s.%s", dep.Namespace, dep.Name)
				}
				if nxa.LocalPort == "" {
					nxa.LocalPort = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
					logrus.Warnf("Fixing w/o local port: %s.%s", dep.Namespace, dep.Name)

				}

				if nxa.Direction == "" {
					nxa.Direction = types.BridgeForward
				}

				ep := &pb.Bridge{}
				nxa.ToPb(ep)
				ep.K8Snamespace = dep.Namespace
				ep.K8Sname = dep.Name
				ep.K8Skind = dep.Kind

				logrus.Infof("K8S Event %v for %s.%s", evt, dep.Name, dep.Namespace)
				switch evt {
				case watch.Added:
					grpc.Server().AddEp(ep)
				case watch.Deleted:
					grpc.Server().RemEp(ep)
				case watch.Modified:
					grpc.Server().AddEp(ep)
				}
			}
		} else if dep.Annotations["nx"] == "" && all {
			ep := &pb.Bridge{}
			ep.Name = dep.Name
			ep.Remoteaddr = dep.Spec.ClusterIP
			ep.Remoteport = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
			ep.Localaddr = dep.Name
			ep.Localport = fmt.Sprintf("%v", dep.Spec.Ports[0].Port)
			ep.Direction = types.BridgeForward
			ep.K8Snamespace = dep.Namespace
			ep.K8Sname = dep.Name
			ep.K8Skind = dep.Kind

			logrus.Infof("K8S Event %v for %s.%s", evt, dep.Name, dep.Namespace)
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

	logrus.Debugf("Loading resources")

	wservices, err := cli.CoreV1().Services(ns).Watch(ctx, v1.ListOptions{})

	if err != nil {
		return err
	}

	go func() {
		for {
			select {

			case x := <-wservices.ResultChan():
				p, ok := x.Object.(*corev1.Service)
				if ok && p != nil {
					handleService(x.Type, p)
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	logrus.Debugf("Namespace monitoring on")
	return nil

}
