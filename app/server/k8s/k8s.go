// package k8s provides support for connecting, watching and creating abstractions
// in kubernetes.
package k8s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// Namespace reads the k8s namespace file if it exists.
func Namespace() string {
	ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return ""
	}

	return string(ns)
}

// =============================================================================

// Config represents settings needed to start the monitor.
type Config struct {
	KubeFile    string
	Namespaces  []string
	AllServices bool
}

// K8s represents an API for working with Kubernetes.
type K8s struct {
	log      *logrus.Logger
	server   *grpc.Server
	wg       sync.WaitGroup
	shutdown context.CancelFunc
}

// Start starts the k8s support for connecting, watching and creating abstractions
// in kubernetes.
func Start(log *logrus.Logger, server *grpc.Server, cfg Config) (*K8s, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	k8s := K8s{
		log:      log,
		server:   server,
		shutdown: cancel,
	}

	restCfg, err := resolveConfig(log, cfg.KubeFile)
	if err != nil {
		return nil, fmt.Errorf("resolveConfig: %w", err)
	}

	log.Infof("k8s: username %q", restCfg.Username)

	clientSet, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("kubernetes.NewForConfig: %w", err)
	}

	k8s.wg.Add(len(cfg.Namespaces))

	for _, ns := range cfg.Namespaces {
		go func(namespace string) {
			log.Infof("k8s: monitor: started: %q", namespace)
			defer func() {
				log.Infof("k8s: monitor: shutdown: %q", namespace)
				k8s.wg.Done()
			}()

			if err := monitor(ctx, log, server, clientSet, namespace, cfg.AllServices); err != nil {
				log.Infof("k8s: monitor: ERROR: %w", err)
			}
		}(ns)
	}

	return &k8s, nil
}

// Shutdown will request all the monitoring G's to shutdown and will wait for
// that to happen.
func (k8s *K8s) Shutdown() {
	k8s.shutdown()
	k8s.wg.Done()
}

// =============================================================================

func resolveConfig(log *logrus.Logger, kubeFile string) (*rest.Config, error) {
	if kubeFile != "" {
		userHomeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("error getting user home dir: %w", err)
		}

		switch {
		case kubeFile == "~":
			kubeFile = filepath.Join(userHomeDir, ".kube", "config")

		case strings.HasPrefix(kubeFile, "~"):
			if len(kubeFile) < 3 {
				return nil, fmt.Errorf("invalid kubeFile %q: %w", kubeFile, err)
			}

			kubeFile = kubeFile[2:]
			kubeFile = filepath.Join(userHomeDir, kubeFile)
		}

		log.Infof("k8s: resolveConfig: using BuildConfigFromFlags %s", kubeFile)
		return clientcmd.BuildConfigFromFlags("", kubeFile)
	}

	log.Infof("k8s: resolveConfig: using InClusterConfig")
	return rest.InClusterConfig()
}

func monitor(ctx context.Context, log *logrus.Logger, server *grpc.Server, clientSet *kubernetes.Clientset, namespace string, allServices bool) error {
	log.Infof("k8s: monitor: started")
	defer log.Infof("k8s: monitor: shutdown")

	pods, err := clientSet.CoreV1().Pods(namespace).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("pods.watch: namespace[%s]: %w", namespace, err)
	}

	services, err := clientSet.CoreV1().Services(namespace).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("services.watch: namespace[%s]: %w", namespace, err)
	}

	deployments, err := clientSet.AppsV1().Deployments(namespace).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("deployments.watch: namespace[%s]: %w", namespace, err)
	}

	statefulSets, err := clientSet.AppsV1().StatefulSets(namespace).Watch(ctx, v1.ListOptions{})
	if err != nil {
		return fmt.Errorf("statefulsets.watch: namespace[%s]: %w", namespace, err)
	}

	h := handlers{
		log:    log,
		server: server,
	}

	for {
		select {
		case x := <-pods.ResultChan():
			pod, ok := x.Object.(*corev1.Pod)
			if ok && pod != nil {
				h.pod(ctx, x.Type, pod)
			}

		case x := <-services.ResultChan():
			service, ok := x.Object.(*corev1.Service)
			if ok && service != nil {
				h.service(ctx, x.Type, service, allServices)
			}

		case x := <-deployments.ResultChan():
			deployment, ok := x.Object.(*appv1.Deployment)
			if ok && deployment != nil {
				h.deployment(ctx, x.Type, deployment)
			}

		case x := <-statefulSets.ResultChan():
			statefulSet, ok := x.Object.(*appv1.StatefulSet)
			if ok && statefulSet != nil {
				h.statefulSets(ctx, x.Type, statefulSet)
			}

		case <-ctx.Done():
			return nil
		}
	}
}

// =============================================================================

type handlers struct {
	log    *logrus.Logger
	server *grpc.Server
}

func (h *handlers) pod(ctx context.Context, evtType watch.EventType, pod *corev1.Pod) error {
	if pod.Annotations["nx"] == "" {
		return nil
	}

	brds, err := bridge.LoadBridges(pod.Annotations["nx"])
	if err != nil {
		return fmt.Errorf("reading annotation for name[%s] namespace[%s]: %w", pod.Name, pod.Namespace, err)
	}

	for i := range brds {
		brd := brds[i]

		if brd.RemoteHost == "" {
			brd.RemoteHost = pod.Status.PodIP
			h.log.Infof("pod: updated RemoteHost: brd.RemoteHost[%s]", brd.RemoteHost)
		}

		if brd.RemotePort == "" {
			brd.RemotePort = fmt.Sprintf("%v", pod.Spec.Containers[0].Ports[0].ContainerPort)
			h.log.Infof("pod: updated RemotePort: brd.RemotePort[%s]", brd.RemotePort)
		}

		prxBrd := bridge.ToProtoBufBridge(brd)
		prxBrd.K8Snamespace = pod.Namespace
		prxBrd.K8Sname = pod.Name
		prxBrd.K8Skind = pod.Kind

		switch evtType {
		case watch.Added:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("pod: added proxy bridge: brd.Name[%s]", brd.Name)

		case watch.Deleted:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("pod: delete proxy bridge: brd.Name[%s]", brd.Name)

		case watch.Modified:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("pod: modified proxy bridge: brd.Name[%s]", brd.Name)
		}
	}

	return nil
}

func (h *handlers) service(ctx context.Context, evtType watch.EventType, service *corev1.Service, all bool) error {
	if service.Annotations["nx"] == "" && !all {
		return nil
	}

	var brds bridge.Bridges
	if service.Annotations["nx"] == "" {
		if !all {
			return nil
		}

		brd := bridge.Bridge{
			Name:      service.Name,
			Direction: bridge.DirectionForward,
		}

		brds = append(brds, brd)
	}

	if len(brds) == 0 {
		var err error
		brds, err = bridge.LoadBridges(service.Annotations["nx"])
		if err != nil {
			return fmt.Errorf("reading annotation for name[%s] namespace[%s]: %w", service.Name, service.Namespace, err)
		}
	}

	for i := range brds {
		brd := brds[i]

		if brd.Name == "" {
			return errors.New("missing bridge name")
		}

		if brd.RemoteHost == "" {
			brd.RemoteHost = service.Spec.ClusterIP
			h.log.Infof("service: updated RemoteHost: brd.RemoteHost[%s]", brd.RemoteHost)
		}

		if brd.RemotePort == "" {
			brd.RemotePort = fmt.Sprintf("%v", service.Spec.Ports[0].Port)
			h.log.Infof("service: updated RemotePort: brd.RemotePort[%s]", brd.RemotePort)
		}

		if brd.LocalPort == "" {
			brd.LocalPort = fmt.Sprintf("%v", service.Spec.Ports[0].Port)
			h.log.Infof("service: updated LocalPort: brd.LocalPort[%s]", brd.LocalPort)
		}

		if brd.Direction == "" {
			brd.Direction = bridge.DirectionReward
			h.log.Infof("service: updated Direction: brd.Direction[%s]", brd.Direction)
		}

		prxBrd := bridge.ToProtoBufBridge(brd)
		prxBrd.K8Snamespace = service.Namespace
		prxBrd.K8Sname = service.Name
		prxBrd.K8Skind = service.Kind

		switch evtType {
		case watch.Added:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("service: added proxy bridge: prxBrd.Name[%s]", prxBrd.Name)

		case watch.Deleted:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("service: delete proxy bridge: prxBrd.Name[%s]", prxBrd.Name)

		case watch.Modified:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("service: modified proxy bridge: prxBrd.Name[%s]", prxBrd.Name)
		}
	}

	return nil
}

func (h *handlers) deployment(ctx context.Context, evtType watch.EventType, deployment *appv1.Deployment) error {
	if deployment.Annotations["nx"] == "" {
		return nil
	}

	brds, err := bridge.LoadBridges(deployment.Annotations["nx"])
	if err != nil {
		return fmt.Errorf("reading annotation for name[%s] namespace[%s]: %w", deployment.Name, deployment.Namespace, err)
	}

	for i := range brds {
		brd := brds[i]

		if brd.Name == "" {
			brd.RemoteHost = fmt.Sprintf("%s.%s", deployment.Namespace, deployment.Name)
			h.log.Infof("deployment: updated RemoteHost: brd.RemoteHost[%s]", brd.RemoteHost)
		}

		if brd.RemoteHost == "" {
			brd.RemoteHost = fmt.Sprintf("%s.%s", deployment.Namespace, deployment.Name)
			h.log.Infof("deployment: updated RemoteHost: brd.RemoteHost[%s]", brd.RemoteHost)
		}

		if brd.LocalHost == "" {
			brd.LocalHost = fmt.Sprintf("%s.%s", deployment.Namespace, deployment.Name)
			h.log.Infof("deployment: updated LocalHost: brd.LocalHost[%s]", brd.LocalHost)
		}

		if brd.RemotePort == "" {
			brd.RemotePort = fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
			h.log.Infof("deployment: updated RemotePort: brd.RemotePort[%s]", brd.RemotePort)
		}

		if brd.LocalPort == "" {
			brd.LocalPort = fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
			h.log.Infof("deployment: updated LocalPort: brd.LocalPort[%s]", brd.LocalPort)
		}

		if brd.Direction == "" {
			brd.Direction = bridge.DirectionForward
			h.log.Infof("deployment: updated Direction: brd.Direction[%s]", brd.Direction)
		}

		prxBrd := bridge.ToProtoBufBridge(brd)
		prxBrd.K8Snamespace = deployment.Namespace
		prxBrd.K8Sname = deployment.Name
		prxBrd.K8Skind = deployment.Kind

		switch evtType {
		case watch.Added:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("deployment: added proxy bridge: brd.Name[%s]", brd.Name)

		case watch.Deleted:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("deployment: delete proxy bridge: brd.Name[%s]", brd.Name)

		case watch.Modified:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("deployment: modified proxy bridge: brd.Name[%s]", brd.Name)
		}
	}

	return nil
}

func (h *handlers) statefulSets(ctx context.Context, evtType watch.EventType, statefulSet *appv1.StatefulSet) error {
	if statefulSet.Annotations["nx"] == "" {
		return nil
	}

	brds, err := bridge.LoadBridges(statefulSet.Annotations["nx"])
	if err != nil {
		return fmt.Errorf("reading annotation for name[%s] namespace[%s]: %w", statefulSet.Name, statefulSet.Namespace, err)
	}

	for i := range brds {
		brd := brds[i]

		if brd.Name == "" {
			brd.RemoteHost = fmt.Sprintf("%s.%s", statefulSet.Namespace, statefulSet.Name)
			h.log.Infof("statefulSets: updated RemoteHost: brd.RemoteHost[%s]", brd.RemoteHost)
		}

		if brd.RemoteHost == "" {
			brd.RemoteHost = fmt.Sprintf("%s.%s", statefulSet.Namespace, statefulSet.Name)
			h.log.Infof("statefulSets: updated RemoteHost: brd.RemoteHost[%s]", brd.RemoteHost)
		}

		if brd.LocalHost == "" {
			brd.LocalHost = fmt.Sprintf("%s.%s", statefulSet.Namespace, statefulSet.Name)
			h.log.Infof("statefulSets: updated LocalHost: brd.LocalHost[%s]", brd.LocalHost)
		}

		if brd.RemotePort == "" {
			brd.RemotePort = fmt.Sprintf("%v", statefulSet.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
			h.log.Infof("statefulSets: updated RemotePort: brd.RemotePort[%s]", brd.RemotePort)
		}

		if brd.LocalPort == "" {
			brd.LocalPort = fmt.Sprintf("%v", statefulSet.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
			h.log.Infof("statefulSets: updated LocalPort: brd.LocalPort[%s]", brd.LocalPort)
		}

		if brd.Direction == "" {
			brd.Direction = bridge.DirectionForward
			h.log.Infof("statefulSets: updated Direction: brd.Direction[%s]", brd.Direction)
		}

		prxBrd := bridge.ToProtoBufBridge(brd)
		prxBrd.K8Snamespace = statefulSet.Namespace
		prxBrd.K8Sname = statefulSet.Name
		prxBrd.K8Skind = statefulSet.Kind

		switch evtType {
		case watch.Added:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("statefulSets: added proxy bridge: brd.Name[%s]", brd.Name)

		case watch.Deleted:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("statefulSets: delete proxy bridge: brd.Name[%s]", brd.Name)

		case watch.Modified:
			h.server.AddProxyBridge(prxBrd)
			h.log.Infof("statefulSets: modified proxy bridge: brd.Name[%s]", brd.Name)
		}
	}

	return nil
}
