package htoken

import (
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsecurity/xrsa"
	"github.com/DreamvatLab/host"
	"github.com/DreamvatLab/oauth2go"
	"github.com/DreamvatLab/oauth2go/security"
	"github.com/DreamvatLab/oauth2go/security/rsa"
	"github.com/DreamvatLab/oauth2go/store/redis"
)

type IOAuthTokenHost interface {
	host.IBaseHost
	host.IWebHost
	// host.ISecureCookieHost
	GetAuthCookieName() string
	GetAuthorizeEndpoint() string
	GetTokenEndpoint() string
	GetEndSessionEndpoint() string
	GetLoginEndpoint() string
	GetLogoutEndpoint() string
}

type OAuthTokenHost struct {
	host.BaseHost
	oauth2go.TokenHost
	host.SecureCookieHost
	UserJsonSessionKey string
	UserIDSessionKey   string
	PrivateKeyPath     string
	ClientStoreKey     string
	TokenStoreKey      string
	SecretEncryptor    security.ISecretEncryptor
}

func (x *OAuthTokenHost) BuildOAuthTokenHost() {
	// slog.Info(x.SecureCookieHost.GetEncryptedCooke)

	x.BaseHost.BuildBaseHost()

	if x.PrivateKeyPath == "" {
		xlog.Fatal("missing 'PrivateKeyPath' filed in configuration")
	}
	if x.UserJsonSessionKey == "" {
		x.UserJsonSessionKey = "USERJSON"
	}
	if x.UserIDSessionKey == "" {
		x.UserIDSessionKey = "USERID"
	}
	if x.ClientStoreKey == "" {
		x.ClientStoreKey = "CLIENTS"
	}
	if x.TokenStoreKey == "" {
		x.ClientStoreKey = "t:"
	}

	////////// CookieEncryptor
	if x.CookieEncryptor == nil {
		x.SecureCookieHost.BuildSecureCookieHost()
		x.CookieEncryptor = x.GetCookieEncryptor()
	}

	////////// PrivateKey
	if x.PrivateKey == nil {
		var err error
		x.PrivateKey, err = xrsa.ReadPrivateKeyFromFile(x.PrivateKeyPath)
		xerr.FatalIfErr(err)
	}

	////////// SecretEncryptor
	if x.SecretEncryptor == nil {
		x.SecretEncryptor = rsa.NewRSASecretEncryptor(x.PrivateKeyPath)
	}

	////////// ClientStore
	if x.ClientStore == nil {
		if x.ClientStoreKey == "" {
			xlog.Fatal("ClientStoreKey cannot be empty")
		}
		if x.RedisConfig == nil {
			xlog.Fatal("missing 'Redis' section in configuration")
		}
		x.ClientStore = redis.NewRedisClientStore(x.ClientStoreKey, x.SecretEncryptor, x.RedisConfig)
	}
	////////// TokenStore
	if x.TokenStore == nil {
		if x.TokenStoreKey == "" {
			xlog.Fatal("TokenStoreKey cannot be empty")
		}
		if x.RedisConfig == nil {
			xlog.Fatal("missing 'Redis' section in configuration")
		}
		x.TokenStore = redis.NewRedisTokenStore(x.TokenStoreKey, x.SecretEncryptor, x.RedisConfig)
	}

	x.TokenHost.BuildTokenHost()
}
