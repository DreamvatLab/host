package hfasthttp

import (
	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/host/hresource"
)

type ResourceHostOption func(*FHOAuthResourceHost)

type FHOAuthResourceHost struct {
	hresource.OAuthResourceHost
	FHWebHost
}

func NewFHOAuthResourceHost(cp xconfig.IConfigProvider, options ...ResourceHostOption) hresource.IOAuthResourceHost {
	r := new(FHOAuthResourceHost)
	// r.OAuthResourceHost = new(resource.OAuthResourceHost)
	// r.OAuthResourceHost.BaseHost = new(host.BaseHost)
	// r.FHWebHost = new(FHWebHost)
	cp.GetStruct("@this", &r)
	r.ConfigProvider = cp

	for _, o := range options {
		o(r)
	}

	r.BuildFHOAuthResourceHost()

	return r
}

func (x *FHOAuthResourceHost) BuildFHOAuthResourceHost() {
	x.BuildOAuthResourceHost()
	x.FHWebHost.buildFHWebHost()
}
