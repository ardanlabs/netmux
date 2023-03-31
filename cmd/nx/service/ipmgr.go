package service

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"go.digitalcircle.com.br/dc/netmux/lib/cmd"
	"sync"
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
	err := cmd.IfconfigRemAlias(Default().Iface, s)
	if err != nil {
		logrus.Warnf("IpMgr.Deallocate::error deallocating ip: %s", err.Error())
	}
}

var ipMgr = &IpMgr{}
