package service

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/foundation/shell"
)

type IpMgr struct {
	ip int
	mx sync.Mutex
}

func (i *IpMgr) Allocate() string {
	i.mx.Lock()
	defer i.mx.Unlock()
	i.ip++
	ipAddrStr := fmt.Sprintf(Default().IpAliasMask, i.ip)
	return ipAddrStr
}

func (i *IpMgr) Deallocate(s string) {
	i.mx.Lock()
	defer i.mx.Unlock()
	err := shell.Ifconfig.RemoveAlias(Default().Iface, s)
	if err != nil {
		logrus.Warnf("IpMgr.Deallocate::error deallocating ip: %s", err.Error())
	}
}

var ipMgr = &IpMgr{}
