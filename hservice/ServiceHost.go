package hservice

import (
	"strings"

	"github.com/DreamvatLab/go/xconv"
	"github.com/DreamvatLab/host"
)

type IServiceHost interface {
	host.IHost
	host.IBaseHost
	GetListenAddr() string
	GetHost() string
	GetPort() int
}

type ServiceHost struct {
	host.BaseHost
	ListenAddr string
	Host       string
	Port       int
}

func (x *ServiceHost) BuildServiceHost() {
	x.BaseHost.BuildBaseHost()
}

func (x *ServiceHost) GetListenAddr() string {
	return x.ListenAddr
}

func (x *ServiceHost) GetHost() string {
	if x.Host == "" {
		a := strings.Split(x.ListenAddr, ":")
		if len(a) == 2 {
			x.Host = a[0]
		}
	}

	return x.Host
}

func (x *ServiceHost) GetPort() int {
	if x.Port <= 0 {
		a := strings.Split(x.ListenAddr, ":")
		if len(a) == 2 {
			x.Port = xconv.ToInt(a[1])
		}
	}

	return x.Port
}
