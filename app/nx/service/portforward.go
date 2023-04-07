package service

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PortForwardAPodRequest struct {
	// RestConfig is the kubernetes config
	RestConfig *rest.Config
	// Pod is the selected pod for this port forwarding
	Pod v1.Pod
	// LocalPort is the local port that will be selected to expose the PodPort
	LocalPort int
	// PodPort is the target port for the pod
	PodPort int
	// Steams configures where to write or read input from
	Streams genericclioptions.IOStreams
	// StopCh is the channel used to manage the port forward lifecycle
	StopCh <-chan struct{}
	// ReadyCh communicates when the tunnel is ready to receive traffic
	ReadyCh chan struct{}
}

// resolveClientConfig will retrive a restconfig, but considering files with
// multiple contexts also.
func resolveClientConfig(configFile string, context string) (*rest.Config, error) {
	configLoadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: configFile}
	configOverrides := &clientcmd.ConfigOverrides{CurrentContext: context}

	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(configLoadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}

// PFStart will start a portforwad from the config file and context.
// This is an in process abstraction, not requiring external processes
func PFStart(kubeconfig string, ctx string, pfreq *PortForwardAPodRequest) (chan struct{}, error) {

	// use the current context in kubeconfig
	config, err := resolveClientConfig(kubeconfig, ctx)
	if err != nil {
		return nil, err
	}
	pfreq.RestConfig = config

	// stopCh control the port forwarding lifecycle. When it gets closed the
	// port forward will terminate
	stopCh := make(chan struct{}, 1)
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})
	errCh := make(chan error)
	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed. For the port forwarding we just need
	// the output eventually
	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    logrus.StandardLogger().Out,
		ErrOut: logrus.StandardLogger().Out,
	}

	pfreq.Streams = stream
	pfreq.ReadyCh = readyCh
	pfreq.StopCh = stopCh
	err = checkPodOrService(pfreq)
	if err != nil {
		return nil, err
	}
	go func() {
		err := PortForwardAPod(pfreq)
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err != nil {
			logrus.Warnf(err.Error())
			return nil, err
		}
	case <-time.After(time.Second * 5):
	case <-readyCh:
	}
	logrus.Infof("Port forwarding is ready to get traffic. have fun!")

	return stopCh, nil
}

func checkPodOrService(req *PortForwardAPodRequest) error {
	clientset, err := kubernetes.NewForConfig(req.RestConfig)
	if err != nil {
		return err
	}

	sl, err := clientset.CoreV1().Endpoints(req.Pod.Namespace).List(context.Background(), metav1.ListOptions{
		FieldSelector: "metadata.name=" + req.Pod.Name,
	})
	if err != nil {
		return err
	}
	if len(sl.Items) > 0 && len(sl.Items[0].Subsets) > 0 {
		ipadddr := sl.Items[0].Subsets[0].Addresses[0].IP

		pods, err := clientset.CoreV1().Pods(req.Pod.Namespace).List(context.Background(), metav1.ListOptions{
			FieldSelector: "status.podIP=" + ipadddr,
		})
		if err != nil {
			return err
		}

		req.Pod.Name = pods.Items[0].Name
	}

	return nil
}

// PortForwardAPod wil effectively do the port forward but to a pod.
// If the PF is expected to be closed w a service, please consider
// PFStart, it will resolve a pod from service and "pretend" PF
// is being oriented to a service.
func PortForwardAPod(req *PortForwardAPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.Pod.Namespace, req.Pod.Name)
	hostIP := strings.TrimLeft(req.RestConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.RestConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, &url.URL{Scheme: "https", Path: path, Host: hostIP})
	fw, err := portforward.New(dialer, []string{fmt.Sprintf("%d:%d", req.LocalPort, req.PodPort)}, req.StopCh, req.ReadyCh, req.Streams.Out, req.Streams.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}
