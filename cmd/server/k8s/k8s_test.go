package k8s_test

import (
	"context"
	"go.digitalcircle.com.br/dc/netmux/cmd/server/grpc"
	"go.digitalcircle.com.br/dc/netmux/cmd/server/k8s"
	"testing"
)

func TestRun(t *testing.T) {
	err := k8s.Run(context.Background(), &k8s.Opts{
		Kubefile:   "~/.kube/k8s.yaml",
		Namespaces: []string{"default"},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestRunDevSimple(t *testing.T) {
	go func() {
		err := grpc.Run()
		if err != nil {
			panic(err)
		}
	}()
	err := k8s.RunDev(context.Background(), &k8s.Opts{
		Kubefile:   "~/.kube/k8s.yaml",
		Namespaces: []string{"default"},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestEvents(t *testing.T) {

}
