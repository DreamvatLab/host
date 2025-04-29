package hfasthttp

import (
	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/host/hclient"
)

type ClientHostOption func(*FHOAuthClientHost)

type FHOAuthClientHost struct {
	hclient.OAuthClientHost
	FHWebHost
}

func NewFHOAuthClientHost(cp xconfig.IConfigProvider, options ...ClientHostOption) hclient.IOAuthClientHost {
	x := new(FHOAuthClientHost)
	cp.GetStruct("@this", &x)
	x.ConfigProvider = cp

	for _, o := range options {
		o(x)
	}

	x.BuildFHOAuthClientHost()

	return x
}

func (x *FHOAuthClientHost) BuildFHOAuthClientHost() {
	x.BuildOAuthClientHost()
	x.FHWebHost.CookieEncryptor = x.SecureCookieHost.GetCookieEncryptor()
	x.FHWebHost.buildFHWebHost()

	////////// oauth client endpoints
	x.Router.GET(x.SignInPath, x.FHWebHost.BuildNativeHandler(x.SignInPath, x.OAuthClientHandler.SignInHandler))
	x.Router.GET(x.SignInCallbackPath, x.FHWebHost.BuildNativeHandler(x.SignInPath, x.OAuthClientHandler.SignInCallbackHandler))
	x.Router.GET(x.SignOutPath, x.FHWebHost.BuildNativeHandler(x.SignInPath, x.OAuthClientHandler.SignOutHandler))
	x.Router.GET(x.SignOutCallbackPath, x.FHWebHost.BuildNativeHandler(x.SignInPath, x.OAuthClientHandler.SignOutCallbackHandler))
}
