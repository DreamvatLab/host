package host

import (
	"embed"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xredis"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/host/hurl"
	"golang.org/x/oauth2"
)

type (
	IOAuthClientHandler interface {
		SignInHandler(ctx IHttpContext)
		SignInCallbackHandler(ctx IHttpContext)
		SignOutHandler(ctx IHttpContext)
		SignOutCallbackHandler(ctx IHttpContext)
	}

	IHost interface {
		Run() error
	}

	IBaseHost interface {
		GetDebug() bool
		GetConfigProvider() xconfig.IConfigProvider
		GetRedisConfig() *xredis.RedisConfig
		GetURLProvider() hurl.IURLProvider
		GetPermissionAuditor() xsecurity.IPermissionAuditor
		GetPermissionProvider() xsecurity.IPermissionProvider
		GetRouteProvider() xsecurity.IRouteProvider
	}

	IWebHost interface {
		IHost
		GET(path string, handlers ...RequestHandler)
		POST(path string, handlers ...RequestHandler)
		PUT(path string, handlers ...RequestHandler)
		PATCH(path string, handlers ...RequestHandler)
		DELETE(path string, handlers ...RequestHandler)
		OPTIONS(path string, handlers ...RequestHandler)
		ServeFiles(webPath, physiblePath string)
		ServeEmbedFiles(webPath, physiblePath string, emd embed.FS)
		AddGlobalPreHandlers(toTail bool, handlers ...RequestHandler)
		AppendGlobalSufHandlers(toTail bool, handlers ...RequestHandler)
		AddActionGroups(actionGroups ...*ActionGroup)
		AddActions(actions ...*Action)
		AddAction(route, routeKey string, handlers ...RequestHandler)
		RegisterActionsToRouter(action *Action)
		NewFSHandler(root string, stripSlashes int) RequestHandler
	}

	IHttpContext interface {
		io.Writer
		SetItem(key string, value interface{})
		GetItem(key string) interface{}
		GetItemString(key string) string
		GetItemInt(key string) int
		GetItemInt32(key string) int32
		GetItemInt64(key string) int64

		GetRouteKey() string

		SetCookieKV(key, value string, options ...func(*http.Cookie))
		GetCookieString(key string) string
		SetEncryptedCookieKV(key, value string, options ...func(*http.Cookie))
		GetEncryptedCookieString(key string) string
		RemoveCookie(key string, options ...func(*http.Cookie))

		SetSession(key, value string)
		GetSessionString(key string) string
		RemoveSession(key string)
		EndSession()

		GetFormString(key string) string
		GetFormStringDefault(key, d string) string
		GetFormFile(key string) (*multipart.FileHeader, error)
		GetMultipartForm() (*multipart.Form, error)

		GetBodyString() string
		GetBodyBytes() []byte

		GetParamString(key string) string
		GetParamInt(key string) int
		GetParamInt32(key string) int32
		GetParamInt64(key string) int64

		ReadJSON(objPtr interface{}) error
		ReadQuery(objPtr interface{}) error
		ReadForm(objPtr interface{}) error
		ReadFormMap() (map[string][]string, error)

		GetHeader(key string) string
		SetHeader(key, value string)

		SetStatusCode(statusCode int)
		SetContentType(cType string)
		WriteString(body string) (int, error)
		WriteBytes(body []byte) (int, error)
		WriteJsonBytes(body []byte) (int, error)

		RequestURL() string
		RequestPath() string
		GetRemoteIP() string
		GetRealIP() string

		UserAgent() string

		Redirect(url string, statusCode int)
		CopyBodyAndStatusCode(resp *http.Response)

		Next()
		Reset()
		GetInnerContext() interface{}
	}

	IContextTokenStore interface {
		SaveToken(ctx IHttpContext, token *oauth2.Token) error
		GetToken(ctx IHttpContext) (*oauth2.Token, error)
	}

	// ISecureCookieHost interface {
	// 	GetEncryptedCookie(ctx IHttpContext, name string) string
	// 	SetEncryptedCookie(ctx IHttpContext, key, value string, options ...func(*http.Cookie))
	// }
)
