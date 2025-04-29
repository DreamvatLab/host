package host

import (
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/host/hurl"
	"github.com/DreamvatLab/oauth2go"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

type (
	OAuthOptions struct {
		*oauth2.Config
		PkceRequired       bool
		EndSessionEndpoint string
		SignOutRedirectURL string
		ClientCredential   *oauth2go.ClientCredential
	}
)

func (x *OAuthOptions) BuildOAuthOptions(urlProvider hurl.IURLProvider) {
	if x.Endpoint.AuthURL == "" {
		xlog.Fatal("OAuth.Endpoint.AuthURL cannot be empty")
	}
	if x.Endpoint.TokenURL == "" {
		xlog.Fatal("OAuth.Endpoint.TokenURL cannot be empty")
	}
	if x.RedirectURL == "" {
		xlog.Fatal("OAuth.RedirectURL cannot be empty")
	}
	if x.SignOutRedirectURL == "" {
		xlog.Fatal("OAuth.SignOutRedirectURL cannot be empty")
	}
	if x.EndSessionEndpoint == "" {
		xlog.Fatal("OAuth.EndSessionEndpoint cannot be empty")
	}

	if urlProvider != nil {
		x.Endpoint.AuthURL = urlProvider.RenderURL(x.Endpoint.AuthURL)
		x.Endpoint.TokenURL = urlProvider.RenderURL(x.Endpoint.TokenURL)
		x.EndSessionEndpoint = urlProvider.RenderURL(x.EndSessionEndpoint)
		x.RedirectURL = urlProvider.RenderURL(x.RedirectURL)
		x.SignOutRedirectURL = urlProvider.RenderURL(x.SignOutRedirectURL)
	}

	x.ClientCredential = &oauth2go.ClientCredential{
		Config: &clientcredentials.Config{
			ClientID:     x.ClientID,
			ClientSecret: x.ClientSecret,
			TokenURL:     x.Endpoint.TokenURL,
			Scopes:       x.Scopes,
		},
	}
}
