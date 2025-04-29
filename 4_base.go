package host

import (
	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xredis"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/host/hurl"
	"github.com/gorilla/securecookie"
)

type BaseHost struct {
	Debug              bool
	Name               string
	URIKey             string
	RouteKey           string
	PermissionKey      string
	RedisConfig        *xredis.RedisConfig `json:"Redis,omitempty"`
	ConfigProvider     xconfig.IConfigProvider
	URLProvider        hurl.IURLProvider
	PermissionProvider xsecurity.IPermissionProvider
	RouteProvider      xsecurity.IRouteProvider
	PermissionAuditor  xsecurity.IPermissionAuditor
}

func (x *BaseHost) BuildBaseHost() {
	if x.ConfigProvider == nil {
		x.ConfigProvider = xconfig.NewJsonConfigProvider()
	}

	if x.URLProvider == nil && x.URIKey != "" && x.RedisConfig != nil {
		var err error
		x.URLProvider, err = hurl.NewRedisURLProvider(x.URIKey, x.RedisConfig)
		if err != nil {
			xlog.Fatal("failed to create redis url provider: " + err.Error())
		}
	}

	if x.PermissionProvider == nil && x.PermissionKey != "" && x.RedisConfig != nil {
		x.PermissionProvider = xsecurity.NewRedisPermissionProvider(x.PermissionKey, x.RedisConfig)
	}

	if x.RouteProvider == nil && x.RouteKey != "" && x.RedisConfig != nil {
		x.RouteProvider = xsecurity.NewRedisRouteProvider(x.RouteKey, x.RedisConfig)
	}

	if x.PermissionAuditor == nil && x.PermissionProvider != nil { // RouteProvider can be empty
		x.PermissionAuditor = xsecurity.NewPermissionAuditor(x.PermissionProvider, x.RouteProvider)
	}

	var logConfig *xlog.LogConfig
	err := x.ConfigProvider.GetStruct("Log", &logConfig)
	xerr.FatalIfErr(err)

	xlog.Init(logConfig)

	ConfigHttpClient(x.ConfigProvider)
}

func (x BaseHost) GetDebug() bool {
	return x.Debug
}

func (x BaseHost) GetConfigProvider() xconfig.IConfigProvider {
	return x.ConfigProvider
}
func (x BaseHost) GetRedisConfig() *xredis.RedisConfig {
	return x.RedisConfig
}
func (x BaseHost) GetURLProvider() hurl.IURLProvider {
	return x.URLProvider
}
func (x BaseHost) GetPermissionAuditor() xsecurity.IPermissionAuditor {
	return x.PermissionAuditor
}
func (x BaseHost) GetPermissionProvider() xsecurity.IPermissionProvider {
	return x.PermissionProvider
}
func (x BaseHost) GetRouteProvider() xsecurity.IRouteProvider {
	return x.RouteProvider
}

type BaseWebHost struct {
	// BaseHost
	ListenAddr        string
	CORS              *CORSOptions
	CookieProtector   *securecookie.SecureCookie
	GlobalPreHandlers []RequestHandler
	GlobalSufHandlers []RequestHandler
	Actions           map[string]*Action
}

func (x *BaseWebHost) BuildBaseWebHost() {
	if x.ListenAddr == "" {
		xlog.Fatal("ListenAddr cannot be empty")
	}

	x.Actions = make(map[string]*Action)
}

// AddGlobalPreHandlers adds global pre-middleware, toTail: whether to append to the end of existing global pre-middleware
func (x *BaseWebHost) AddGlobalPreHandlers(toTail bool, handlers ...RequestHandler) {
	if toTail {
		x.GlobalPreHandlers = append(x.GlobalPreHandlers, handlers...)
	} else {
		x.GlobalPreHandlers = append(handlers, x.GlobalPreHandlers...)
	}
}

// AppendGlobalSufHandlers adds global post-middleware, toTail: whether to append to the end of existing global post-middleware
func (x *BaseWebHost) AppendGlobalSufHandlers(toTail bool, handlers ...RequestHandler) {
	if toTail {
		x.GlobalSufHandlers = append(x.GlobalSufHandlers, handlers...)
	} else {
		x.GlobalSufHandlers = append(handlers, x.GlobalSufHandlers...)
	}
}

func (x *BaseWebHost) AddActionGroups(actionGroups ...*ActionGroup) {
	////////// Add Actions
	for _, actionGroup := range actionGroups {
		for _, action := range actionGroup.Actions {
			// Add pre-execution middleware
			if len(actionGroup.PreHandlers) > 0 {
				action.Handlers = append(actionGroup.PreHandlers, action.Handlers...)
			}
			// Add post-execution middleware
			if len(actionGroup.AfterHandlers) > 0 {
				action.Handlers = append(action.Handlers, actionGroup.AfterHandlers...)
			}

			_, ok := x.Actions[action.Route]
			if ok {
				xlog.Fatal("duplicated route found: " + action.Route)
			}
			x.Actions[action.Route] = action
		}
	}
}

func (x *BaseWebHost) AddActions(actions ...*Action) {
	////////// Add Actions
	for _, action := range actions {
		_, ok := x.Actions[action.Route]
		if ok {
			xlog.Fatal("duplicated route found: " + action.Route)
		}
		x.Actions[action.Route] = action
	}
}

func (x *BaseWebHost) AddAction(route, routeKey string, handlers ...RequestHandler) {
	////////// 添加Action
	action := NewAction(route, routeKey, handlers...)
	_, ok := x.Actions[action.Route]
	if ok {
		xlog.Fatal("duplicated route found: " + action.Route)
	}
	x.Actions[action.Route] = action
}

type SecureCookieHost struct {
	HashKey         string
	BlockKey        string
	cookieEncryptor xsecurity.ICookieEncryptor
}

func (x *SecureCookieHost) GetCookieEncryptor() xsecurity.ICookieEncryptor {
	return x.cookieEncryptor
}

func (x *SecureCookieHost) BuildSecureCookieHost() {
	if x.BlockKey == "" {
		xlog.Fatal("block key cannot be empty")
	}
	if x.HashKey == "" {
		xlog.Fatal("hash key cannot be empty")
	}

	x.cookieEncryptor = xsecurity.NewSecureCookieEncryptor(xbytes.StrToBytes(x.HashKey), xbytes.StrToBytes(x.BlockKey))
}
