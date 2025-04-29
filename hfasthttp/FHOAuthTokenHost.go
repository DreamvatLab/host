package hfasthttp

import (
	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/host/htoken"
)

type TokenHostOption func(*FHOAuthTokenHost)

type FHOAuthTokenHost struct {
	htoken.OAuthTokenHost
	FHWebHost
}

func NewFHOAuthTokenHost(cp xconfig.IConfigProvider, options ...TokenHostOption) htoken.IOAuthTokenHost {
	r := new(FHOAuthTokenHost)
	cp.GetStruct("@this", &r)
	r.ConfigProvider = cp

	for _, o := range options {
		o(r)
	}

	r.BuildFHOAuthTokenHost()

	return r
}

func (x *FHOAuthTokenHost) BuildFHOAuthTokenHost() {
	x.BuildOAuthTokenHost()
	x.FHWebHost.CookieEncryptor = x.SecureCookieHost.GetCookieEncryptor()
	x.FHWebHost.buildFHWebHost()

	x.Router.POST(x.TokenEndpoint, x.TokenHost.TokenRequestHandler)
	x.Router.GET(x.AuthorizeEndpoint, x.TokenHost.AuthorizeRequestHandler)
	x.Router.GET(x.EndSessionEndpoint, x.TokenHost.EndSessionRequestHandler)
	x.Router.POST(x.EndSessionEndpoint, x.TokenHost.ClearTokenRequestHandler)
}
