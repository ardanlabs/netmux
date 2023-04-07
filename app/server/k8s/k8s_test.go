package k8s_test

import (
	"context"
	"testing"

	"go.digitalcircle.com.br/dc/netmux/app/server/grpc"
	"go.digitalcircle.com.br/dc/netmux/app/server/k8s"
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
