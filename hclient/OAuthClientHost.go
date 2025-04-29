package hclient

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/host"
	"github.com/muesli/cache2go"
	"golang.org/x/oauth2"
)

type IOAuthClientHost interface {
	host.IBaseHost
	host.IWebHost
	// host.ISecureCookieHost
	AuthHandler(ctx host.IHttpContext)
	GetHttpClient() *http.Client
	GetUserHttpClient(ctx host.IHttpContext) (*http.Client, error)
	GetClientToken(ctx host.IHttpContext) (*oauth2.Token, error)
	GetUserToken(ctx host.IHttpContext) (*oauth2.TokenSource, error)
	GetUserLock(userID string) *sync.RWMutex
	GetUserJsonSessionKey() string
	GetUserIDSessionKey() string
}

type OAuthClientHost struct {
	host.BaseHost
	host.SecureCookieHost
	OAuthOptions        *host.OAuthOptions `json:"OAuth,omitempty"`
	UserJsonSessionKey  string
	UserIDSessionKey    string
	TokenCookieName     string
	SignInPath          string
	SignInCallbackPath  string
	SignOutPath         string
	SignOutCallbackPath string
	AccessDeniedPath    string
	OAuthClientHandler  host.IOAuthClientHandler
	ContextTokenStore   host.IContextTokenStore
	UserLocks           *cache2go.CacheTable
	CookieEncryptor     xsecurity.ICookieEncryptor
}

func (x *OAuthClientHost) BuildOAuthClientHost() {
	x.BaseHost.BuildBaseHost()

	if x.OAuthOptions == nil {
		xlog.Fatal("OAuth secion in configuration is missing")
	}
	x.OAuthOptions.BuildOAuthOptions(x.URLProvider)

	if x.SignInPath == "" {
		x.SignInPath = "/signin"
	}
	if x.SignInCallbackPath == "" {
		x.SignInCallbackPath = "/signin-oauth"
	}
	if x.SignOutPath == "" {
		x.SignOutPath = "/signout"
	}
	if x.SignOutCallbackPath == "" {
		x.SignOutCallbackPath = "/signout-oauth"
	}
	if x.AccessDeniedPath == "" {
		x.AccessDeniedPath = "/accessdenied"
	}
	if x.UserJsonSessionKey == "" {
		x.UserJsonSessionKey = "USERJSON"
	}
	if x.UserIDSessionKey == "" {
		x.UserIDSessionKey = "USERID"
	}
	if x.TokenCookieName == "" {
		x.TokenCookieName = "go.cookie2"
	}

	////////// user locks
	if x.UserLocks == nil {
		x.UserLocks = cache2go.Cache("UserLocks")
	}

	////////// CookieEncryptor
	if x.CookieEncryptor == nil {
		x.SecureCookieHost.BuildSecureCookieHost()
		x.CookieEncryptor = x.GetCookieEncryptor()
	}

	////////// context token store
	if x.ContextTokenStore == nil {
		x.ContextTokenStore = NewCookieTokenStore(x.TokenCookieName, x.CookieEncryptor)
	}

	////////// oauth client handler
	if x.OAuthClientHandler == nil {
		x.OAuthClientHandler = NewOAuthClientHandler(x.OAuthOptions, x.ContextTokenStore, x.UserJsonSessionKey, x.UserIDSessionKey, x.TokenCookieName)
	}

	// ////////// auth middleware
	// if x.authMiddleware == nil {
	// 	x.authMiddleware = newClientAuthMiddleware(x.UserJsonSessionKey, x.AccessDeniedPath, x.OAuthOptions, x.PermissionAuditor)
	// }
}

func (x *OAuthClientHost) GetHttpClient() *http.Client {
	return x.OAuthOptions.ClientCredential.Client(context.Background())
}

func (x *OAuthClientHost) GetUserHttpClient(ctx host.IHttpContext) (*http.Client, error) {
	// goctx := context.Background()
	// userID := host.GetUserID(ctx, x.UserIDSessionKey)
	// if userID == "" {
	// 	return http.DefaultClient, nil
	// }

	// // 获取用户锁
	// userLock := x.getUserLock(userID)

	// // read lock
	// userLock.RLock()
	// t, err := x.ContextTokenStore.GetToken(ctx)
	// defer func() { userLock.RUnlock() }()
	// if err != nil {
	// 	return http.DefaultClient, err
	// }

	// tokenSource := x.OAuthOptions.TokenSource(goctx, t)
	// newToken, err := tokenSource.Token()
	// if err != nil {
	// 	// refresh token failed, sign user out
	// 	host.SignOut(ctx, x.TokenCookieName)
	// 	return http.DefaultClient, err
	// }

	// if newToken.AccessToken != t.AccessToken {
	// 	// token been refreshed, lock
	// 	userLock.Lock()
	// 	// save token to session
	// 	err = x.ContextTokenStore.SaveToken(ctx, newToken)
	// 	// unlock
	// 	defer func() { userLock.Unlock() }()
	// 	if err != nil {
	// 		return http.DefaultClient, err
	// 	}
	// }

	tokenSource, err := x.GetUserToken(ctx)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(context.Background(), *tokenSource), nil
}

func (x *OAuthClientHost) GetClientToken(ctx host.IHttpContext) (*oauth2.Token, error) {
	return x.OAuthOptions.ClientCredential.Token()
}

func (x *OAuthClientHost) GetUserToken(ctx host.IHttpContext) (*oauth2.TokenSource, error) {
	goctx := context.Background()
	userID := host.GetUserID(ctx, x.UserIDSessionKey)
	if userID == "" {
		return nil, xerr.New("user isn't authenticated")
	}

	// Get user lock
	userLock := x.GetUserLock(userID)

	// read lock
	userLock.RLock()
	t, err := x.ContextTokenStore.GetToken(ctx)
	if err != nil {
		userLock.RUnlock() // read unlock
		return nil, xerr.WithStack(err)
	}
	userLock.RUnlock() // read unlock

	tokenSource := x.OAuthOptions.TokenSource(goctx, t)
	newToken, err := tokenSource.Token()
	if err != nil {
		// refresh token failed, sign user out
		host.SignOut(ctx, x.TokenCookieName)
		return nil, xerr.WithStack(err)
	}

	if newToken.AccessToken != t.AccessToken {
		// token been refreshed, lock
		userLock.Lock()
		// save token to session
		err = x.ContextTokenStore.SaveToken(ctx, newToken)
		// unlock
		userLock.Unlock()
		if err != nil {
			return nil, err
		}
	}

	return &tokenSource, nil
}

func (x *OAuthClientHost) AuthHandler(ctx host.IHttpContext) {
	routeKey := ctx.GetItemString(host.Ctx_RouteKey)
	if routeKey == "" {
		ctx.SetStatusCode(500)
		ctx.WriteString("route key does not exist")
		return
	}

	routes := strings.Split(routeKey, "_")

	var area, controller, action string
	count := len(routes)
	if count == 0 || count > 3 {
		xlog.Fatal("invalid routes array")
	}

	area = routes[0]
	if count >= 2 {
		controller = routes[1]
	}
	if count == 3 {
		action = routes[2]
	}

	user := host.GetUser(ctx, x.UserJsonSessionKey)

	// Check if request is allowed to access
	if user != nil {
		if x.PermissionAuditor.CheckRouteWithLevel(area, controller, action, user.Role, user.Level, user.Scopes) {
			// Has permission
			ctx.Next()
			return
		} else {
			// No permission
			ctx.Redirect(x.AccessDeniedPath, http.StatusFound)
			return
		}
	}

	// Not logged in
	allow := x.PermissionAuditor.CheckRouteWithLevel(area, controller, action, 0, 0, make([]string, 0))
	if allow {
		// Allow anonymous
		ctx.Next()
		return
	}

	// 记录请求地址，跳转去登录页面
	host.RedirectAuthorizeEndpoint(ctx, x.OAuthOptions, ctx.RequestURL())
}

func (x *OAuthClientHost) GetUserLock(userID string) *sync.RWMutex {
	if !x.UserLocks.Exists(userID) {
		x.UserLocks.Add(userID, time.Second*30, new(sync.RWMutex))
	}

	userLockCache, err := x.UserLocks.Value(userID)
	xerr.LogError(err)
	return userLockCache.Data().(*sync.RWMutex)
}

func (x *OAuthClientHost) GetUserJsonSessionKey() string {
	return x.UserJsonSessionKey
}

func (x *OAuthClientHost) GetUserIDSessionKey() string {
	return x.UserIDSessionKey
}
