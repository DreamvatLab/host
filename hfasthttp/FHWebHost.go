package hfasthttp

import (
	"embed"
	"mime"
	"net/http"
	fp "path/filepath"
	"strings"
	"time"

	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/host"
	"github.com/fasthttp/router"
	"github.com/fasthttp/session/v2"
	"github.com/fasthttp/session/v2/providers/memory"
	"github.com/valyala/fasthttp"
)

const (
	_filepath = "filepath"
	_suffix   = "/{" + _filepath + ":*}"
)

type WebHostOption func(*FHWebHost)

// FHWebHost : IWebHost
type FHWebHost struct {
	host.BaseWebHost
	// Unique properties
	IndexName          string
	SessionCookieName  string
	SessionExpSeconds  int
	ReadBufferSize     int
	MaxRequestBodySize int
	Router             *router.Router
	SessionProvider    session.Provider
	SessionManager     *session.Session
	// HTTP request Handler, if specified, the Router's Handler will not be used
	HttpHandler     host.RequestHandler
	PanicHandler    host.RequestHandler
	CookieEncryptor xsecurity.ICookieEncryptor
	fsHandler       fasthttp.RequestHandler
}

func NewFHWebHost(cp xconfig.IConfigProvider, options ...WebHostOption) host.IWebHost {
	r := new(FHWebHost)
	cp.GetStruct("@this", &r)

	for _, o := range options {
		o(r)
	}

	r.buildFHWebHost()

	return r
}

func (x *FHWebHost) buildFHWebHost() {
	x.BuildBaseWebHost()

	if x.IndexName == "" {
		x.IndexName = "index.html"
	}

	if x.SessionCookieName == "" {
		x.SessionCookieName = "go.cookie1"
	}

	////////// router
	if x.Router == nil {
		x.Router = router.New()
		x.Router.PanicHandler = func(ctx *fasthttp.RequestCtx, err interface{}) {
			if x.PanicHandler != nil {
				newCtx := NewFastHttpContext(ctx, x.SessionManager, x.CookieEncryptor)
				newCtx.SetItem(host.Ctx_Panic, err)
				x.PanicHandler(newCtx)
				return
			}
			ctx.SetStatusCode(500)
			xlog.Error(err)
		}
	}

	////////// session provider
	if x.SessionProvider == nil {
		provider, err := memory.New(memory.Config{})
		xerr.FatalIfErr(err)
		x.SessionProvider = provider
	}

	////////// session manager
	if x.SessionManager == nil {
		cfg := session.NewDefaultConfig()
		if x.SessionExpSeconds <= 0 {
			cfg.Expiration = -1
		} else {
			cfg.Expiration = time.Second * time.Duration(x.SessionExpSeconds)
		}
		cfg.CookieName = x.SessionCookieName
		cfg.EncodeFunc = session.MSGPEncode // Memory provider has better performance
		cfg.DecodeFunc = session.MSGPDecode // Memory provider has better performance

		x.SessionManager = session.New(cfg)
		err := x.SessionManager.SetProvider(x.SessionProvider)
		xerr.FatalIfErr(err)
	}

	if x.ReadBufferSize <= 0 {
		x.ReadBufferSize = 4096
	}

	if x.MaxRequestBodySize <= 0 {
		x.MaxRequestBodySize = fasthttp.DefaultMaxRequestBodySize
	}

	////////// CORS
	if x.CORS != nil {
		x.AddGlobalPreHandlers(true, func(ctx host.IHttpContext) {
			if x.CORS.AllowedOrigin != "" {
				ctx.SetHeader("Access-Control-Allow-Origin", x.CORS.AllowedOrigin)
			}
			ctx.Next()
		})

		x.OPTIONS("/{filepath:*}", func(ctx host.IHttpContext) {
			// if x.CORS.AllowedOrigin != "" {	// Already added by global middleware above
			// 	ctx.SetHeader("Access-Control-Allow-Origin", x.CORS.AllowedOrigin)
			// }
			if x.CORS.AllowedMethods != "" {
				ctx.SetHeader("Access-Control-Allow-Methods", x.CORS.AllowedMethods)
			}
			if x.CORS.AllowedHeaders != "" {
				ctx.SetHeader("Access-Control-Allow-Headers", x.CORS.AllowedHeaders)
			}
		})
	}
}

func (x *FHWebHost) BuildNativeHandler(routeKey string, handlers ...host.RequestHandler) fasthttp.RequestHandler {
	if len(handlers) == 0 {
		xlog.Fatal("handlers are missing")
	}

	// Register global middleware
	if len(x.GlobalPreHandlers) > 0 {
		handlers = append(x.GlobalPreHandlers, handlers...)
	}
	if len(x.GlobalSufHandlers) > 0 {
		handlers = append(handlers, x.GlobalSufHandlers...)
	}

	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		newCtx := NewFastHttpContext(ctx, x.SessionManager, x.CookieEncryptor, handlers...)
		newCtx.SetItem(host.Ctx_RouteKey, routeKey)
		defer func() {
			newCtx.Reset()
			_ctxPool.Put(newCtx)
		}()
		handlers[0](newCtx) // Start executing the first Handler
	})
}

func (x *FHWebHost) NewFSHandler(root string, stripSlashes int) host.RequestHandler {
	x.fsHandler = fasthttp.FSHandler(root, stripSlashes)
	return func(ctx host.IHttpContext) {
		c := ctx.GetInnerContext().(*fasthttp.RequestCtx)
		x.fsHandler(c)
	}
}

func (x *FHWebHost) GET(path string, handlers ...host.RequestHandler) {
	x.Router.GET(path, x.BuildNativeHandler(path, handlers...))
}
func (x *FHWebHost) POST(path string, handlers ...host.RequestHandler) {
	x.Router.POST(path, x.BuildNativeHandler(path, handlers...))
}
func (x *FHWebHost) PUT(path string, handlers ...host.RequestHandler) {
	x.Router.PUT(path, x.BuildNativeHandler(path, handlers...))
}
func (x *FHWebHost) PATCH(path string, handlers ...host.RequestHandler) {
	x.Router.PATCH(path, x.BuildNativeHandler(path, handlers...))
}
func (x *FHWebHost) DELETE(path string, handlers ...host.RequestHandler) {
	x.Router.DELETE(path, x.BuildNativeHandler(path, handlers...))
}
func (x *FHWebHost) OPTIONS(path string, handlers ...host.RequestHandler) {
	x.Router.OPTIONS(path, x.BuildNativeHandler(path, handlers...))
}

func (x *FHWebHost) ServeFiles(webPath, physiblePath string) {
	x.Router.ServeFiles(webPath, physiblePath)
}

func (x *FHWebHost) ServeEmbedFiles(webPath, physiblePath string, emd embed.FS) {
	if !strings.HasSuffix(webPath, _suffix) {
		panic("path must end with " + _suffix + " in path '" + webPath + "'")
	}

	x.Router.GET(webPath, func(ctx *fasthttp.RequestCtx) {
		filepath := ctx.UserValue(_filepath).(string)
		if filepath == "" {
			filepath = x.IndexName
		}

		filepath = physiblePath + "/" + filepath

		file, err := emd.Open(filepath) // embed file doesn't need to close
		if err == nil {
			ext := fp.Ext(filepath)
			cType := mime.TypeByExtension(ext)

			if cType != "" {
				ctx.SetContentType(cType)
			}
			ctx.Response.SetBodyStream(file, -1)
			return
		}

		ctx.SetStatusCode(404)
		ctx.WriteString("NOT FOUND")
	})
}

func (x *FHWebHost) Run() error {
	////////// Register Actions to router
	for _, v := range x.Actions {
		x.RegisterActionsToRouter(v)
	}

	////////// Start Serve
	xlog.Infof("Listening on %s", x.ListenAddr)

	var handler fasthttp.RequestHandler
	if x.HttpHandler == nil {
		handler = x.Router.Handler
	} else {
		handler = x.BuildNativeHandler("General", x.HttpHandler)
	}

	s := &fasthttp.Server{
		// Handler:        x.Router.Handler,
		Handler:            handler,
		ReadBufferSize:     x.ReadBufferSize, // Increase this value to resolve Http 431 error
		MaxRequestBodySize: x.MaxRequestBodySize,
		Logger:             new(debugLogger),
	}
	return s.ListenAndServe(x.ListenAddr)
}

func (x *FHWebHost) RegisterActionsToRouter(action *host.Action) {
	index := strings.Index(action.Route, "/")
	method := action.Route[:index]
	path := action.Route[index:]

	switch method {
	case http.MethodPost:
		x.Router.POST(path, x.BuildNativeHandler(action.RouteKey, action.Handlers...))
	case http.MethodGet:
		x.Router.GET(path, x.BuildNativeHandler(action.RouteKey, action.Handlers...))
	case http.MethodPut:
		x.Router.PUT(path, x.BuildNativeHandler(action.RouteKey, action.Handlers...))
	case http.MethodPatch:
		x.Router.PATCH(path, x.BuildNativeHandler(action.RouteKey, action.Handlers...))
	case http.MethodDelete:
		x.Router.DELETE(path, x.BuildNativeHandler(action.RouteKey, action.Handlers...))
	case http.MethodOptions:
		x.Router.OPTIONS(path, x.BuildNativeHandler(action.RouteKey, action.Handlers...))
	default:
		panic("does not support method " + method)
	}
}
