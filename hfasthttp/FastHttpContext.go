package hfasthttp

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"sync"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xconv"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xhttp"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/go/xsync"
	"github.com/DreamvatLab/host"
	"github.com/fasthttp/session/v2"
	"github.com/gorilla/schema"
	"github.com/valyala/fasthttp"
)

var (
	_ctxPool = &sync.Pool{
		New: func() interface{} {
			return new(FastHttpContext)
		},
	}
	_cookiePool = xsync.NewSyncCookiePool()
	_decoder    = schema.NewDecoder()
	// _setCookieKVExpiration = 8760 * time.Hour
)

func init() {
	_decoder.IgnoreUnknownKeys(true)
}

type FastHttpContext struct {
	ctx             *fasthttp.RequestCtx
	sess            *session.Session
	sessStore       *session.Store
	mapPool         *sync.Pool
	cookieEncryptor xsecurity.ICookieEncryptor
	handlers        []host.RequestHandler
	handlerIndex    int
	handlerCount    int
}

func NewFastHttpContext(ctx *fasthttp.RequestCtx, sess *session.Session, cookieEncryptor xsecurity.ICookieEncryptor, handlers ...host.RequestHandler) host.IHttpContext {
	r := _ctxPool.Get().(*FastHttpContext)
	r.ctx = ctx
	r.sess = sess
	r.cookieEncryptor = cookieEncryptor
	var err error
	r.sessStore, err = r.sess.Get(ctx)
	xerr.FatalIfErr(err)
	r.mapPool = &sync.Pool{
		New: func() interface{} {
			return make(map[string][]string)
		},
	}
	r.handlers = handlers
	r.handlerCount = len(handlers)
	return r
}

func (x *FastHttpContext) GetInnerContext() interface{} {
	return x.ctx
}

func (x *FastHttpContext) Write(p []byte) (n int, err error) {
	return x.ctx.Write(p)
}

func (x *FastHttpContext) SetItem(key string, value interface{}) {
	x.ctx.SetUserValue(key, value)
}
func (x *FastHttpContext) GetItem(key string) interface{} {
	return x.ctx.UserValue(key)
}
func (x *FastHttpContext) GetItemString(key string) string {
	v := x.ctx.UserValue(key)
	return xconv.ToString(v)
}
func (x *FastHttpContext) GetItemInt(key string) int {
	v := x.ctx.UserValue(key)
	return xconv.ToInt(v)
}
func (x *FastHttpContext) GetItemInt32(key string) int32 {
	v := x.ctx.UserValue(key)
	return xconv.ToInt32(v)
}
func (x *FastHttpContext) GetItemInt64(key string) int64 {
	v := x.ctx.UserValue(key)
	return xconv.ToInt64(v)
}

func (x *FastHttpContext) GetRouteKey() string {
	return x.GetItemString(host.Ctx_RouteKey)
}

func (x *FastHttpContext) setCookie(cookie *http.Cookie) {
	c := fasthttp.AcquireCookie()
	defer func() {
		fasthttp.ReleaseCookie(c)
	}()

	c.SetKey(cookie.Name)
	c.SetValue(cookie.Value)
	c.SetDomain(cookie.Domain)
	c.SetPath(cookie.Path)
	c.SetSecure(cookie.Secure)
	c.SetHTTPOnly(cookie.HttpOnly)
	c.SetExpire(cookie.Expires)
	c.SetMaxAge(cookie.MaxAge)
	x.ctx.Response.Header.SetCookie(c)
}
func (x *FastHttpContext) SetCookieKV(key, value string, options ...func(*http.Cookie)) {
	c := _cookiePool.GetCookie()
	defer func() {
		_cookiePool.PutCookie(c)
	}()

	c.Name = key
	c.Value = value

	for _, o := range options {
		o(c)
	}

	x.setCookie(c)
}
func (x *FastHttpContext) GetCookieString(key string) string {
	r := x.ctx.Request.Header.Cookie(key)
	return xbytes.BytesToStr(r)
}

func (x *FastHttpContext) SetEncryptedCookieKV(key, value string, options ...func(*http.Cookie)) {
	if x.cookieEncryptor == nil {
		xlog.Warn("cookieEncryptor is nil, this context does not suppot cookie encryption")
		return
	}
	encryptedString, err := x.cookieEncryptor.Encrypt(key, value)
	if xerr.LogError(err) {
		return
	}

	x.SetCookieKV(key, encryptedString, options...)
}

func (x *FastHttpContext) GetEncryptedCookieString(key string) (r string) {
	if x.cookieEncryptor == nil {
		xlog.Warn("cookieEncryptor is nil, this context does not suppot cookie encryption")
		return
	}

	encryptedString := x.GetCookieString(key)
	if encryptedString != "" {
		err := x.cookieEncryptor.Decrypt(key, encryptedString, &r)
		xerr.LogError(err)
	}

	return
}

func (x *FastHttpContext) RemoveCookie(key string, options ...func(*http.Cookie)) {
	if len(options) > 0 {
		x.ctx.Response.Header.DelCookie(key)

		c := _cookiePool.GetCookie()
		defer func() {
			_cookiePool.PutCookie(c)
		}()
		c.Name = key
		c.Expires = fasthttp.CookieExpireDelete

		for _, o := range options {
			o(c)
		}

		x.setCookie(c)
	} else {
		x.ctx.Response.Header.DelClientCookie(key)
	}
}

func (x *FastHttpContext) SetSession(key, value string) {
	store, err := x.sess.Get(x.ctx)
	if xerr.LogError(err) {
		return
	}
	defer func() {
		xerr.LogError(x.sess.Save(x.ctx, store))
	}()
	store.Set(key, value)
}
func (x *FastHttpContext) GetSessionString(key string) string {
	store, err := x.sess.Get(x.ctx)
	if xerr.LogError(err) {
		return ""
	}
	defer func() {
		xerr.LogError(x.sess.Save(x.ctx, store))
	}()

	if r, ok := store.Get(key).(string); ok {
		return r
	}

	return ""
}
func (x *FastHttpContext) RemoveSession(key string) {
	store, err := x.sess.Get(x.ctx)
	if xerr.LogError(err) {
		return
	}
	defer func() {
		xerr.LogError(x.sess.Save(x.ctx, store))
	}()
	store.Delete(key)
}
func (x *FastHttpContext) EndSession() {
	x.sess.Destroy(x.ctx)
}

func (x *FastHttpContext) GetFormString(key string) string {
	r := x.ctx.FormValue(key)
	return xbytes.BytesToStr(r)
}
func (x *FastHttpContext) GetFormStringDefault(key, d string) (r string) {
	data := x.ctx.FormValue(key)
	r = xbytes.BytesToStr(data)
	if r == "" {
		r = d
	}
	return
}

func (x *FastHttpContext) GetFormFile(key string) (*multipart.FileHeader, error) {
	r, err := x.ctx.FormFile(key)
	return r, xerr.WithStack(err)
}

func (x *FastHttpContext) GetMultipartForm() (*multipart.Form, error) {
	r, err := x.ctx.MultipartForm()
	return r, xerr.WithStack(err)
}

func (x *FastHttpContext) GetBodyString() string {
	// return x.ctx.Request.String()
	return xbytes.BytesToStr(x.ctx.Request.Body())
}
func (x *FastHttpContext) GetBodyBytes() []byte {
	return x.ctx.Request.Body()
}

func (x *FastHttpContext) GetParamString(key string) string {
	v := x.ctx.UserValue(key)
	return xconv.ToString(v)
}
func (x *FastHttpContext) GetParamInt(key string) int {
	v := x.ctx.UserValue(key)
	return xconv.ToInt(v)
}
func (x *FastHttpContext) GetParamInt32(key string) int32 {
	v := x.ctx.UserValue(key)
	return xconv.ToInt32(v)
}
func (x *FastHttpContext) GetParamInt64(key string) int64 {
	v := x.ctx.UserValue(key)
	return xconv.ToInt64(v)
}

func (x *FastHttpContext) ReadJSON(objPtr interface{}) error {
	data := x.ctx.Request.Body()
	err := json.Unmarshal(data, objPtr)
	return xerr.WithStack(err)
}
func (x *FastHttpContext) ReadQuery(objPtr interface{}) error {
	dic := x.mapPool.Get().(map[string][]string)
	defer func() {
		for k := range dic { // this will compile to use "mapclear" internal function
			delete(dic, k)
		}
		x.mapPool.Put(dic)
	}()
	x.ctx.QueryArgs().VisitAll(func(key, value []byte) {
		dic[xbytes.BytesToStr(key)] = []string{xbytes.BytesToStr(value)}
	})

	err := _decoder.Decode(objPtr, dic)
	return xerr.WithStack(err)
}
func (x *FastHttpContext) ReadForm(objPtr interface{}) error {
	dic := x.mapPool.Get().(map[string][]string)
	defer func() {
		for k := range dic { // this will compile to use "mapclear" internal function
			delete(dic, k)
		}
		x.mapPool.Put(dic)
	}()
	x.ctx.PostArgs().VisitAll(func(key, value []byte) {
		dic[xbytes.BytesToStr(key)] = []string{xbytes.BytesToStr(value)}
	})

	err := _decoder.Decode(objPtr, dic)
	return xerr.WithStack(err)
}

func (x *FastHttpContext) ReadFormMap() (map[string][]string, error) {
	dic := make(map[string][]string)

	x.ctx.PostArgs().VisitAll(func(key, value []byte) {
		dic[xbytes.BytesToStr(key)] = []string{xbytes.BytesToStr(value)}
	})

	return dic, nil
}

func (x *FastHttpContext) SetHeader(key, value string) {
	x.ctx.Response.Header.Set(key, value)
}
func (x *FastHttpContext) GetHeader(key string) string {
	v := x.ctx.Request.Header.Peek(key)
	return xbytes.BytesToStr(v)
}

func (x *FastHttpContext) SetStatusCode(statusCode int) {
	x.ctx.SetStatusCode(statusCode)
}
func (x *FastHttpContext) SetContentType(cType string) {
	x.ctx.SetContentType(cType)
}
func (x *FastHttpContext) WriteString(body string) (int, error) {
	r, err := x.ctx.WriteString(body)
	return r, xerr.WithStack(err)
}
func (x *FastHttpContext) WriteBytes(body []byte) (int, error) {
	r, err := x.ctx.Write(body)
	return r, xerr.WithStack(err)
}

func (x *FastHttpContext) WriteJsonBytes(body []byte) (int, error) {
	x.ctx.SetContentType(xhttp.CTYPE_JSON)
	r, err := x.ctx.Write(body)
	return r, xerr.WithStack(err)
}

func (x *FastHttpContext) RequestURL() string {
	return x.ctx.URI().String()
}
func (x *FastHttpContext) RequestPath() string {
	return xbytes.BytesToStr(x.ctx.URI().Path())
}
func (x *FastHttpContext) GetRemoteIP() string {
	return x.ctx.RemoteIP().String()
}

func (x *FastHttpContext) GetRealIP() string {
	var ips []string

	// 从 X-Forwarded-For 获取所有 IP
	if forwardedFor := x.ctx.Request.Header.Peek("X-Forwarded-For"); len(forwardedFor) > 0 {
		// 分割所有 IP 地址
		ipList := bytes.Split(forwardedFor, []byte(","))
		for _, ip := range ipList {
			// 去除空格
			ip = bytes.TrimSpace(ip)
			if len(ip) > 0 {
				ips = append(ips, string(ip))
			}
		}
	}

	// 如果没有 X-Forwarded-For，则添加 RemoteIP
	if len(ips) == 0 {
		ips = append(ips, x.ctx.RemoteIP().String())
	}

	return strings.Join(ips, "\n")
}

func (x *FastHttpContext) UserAgent() string {
	return xbytes.BytesToStr(x.ctx.UserAgent())
}

func (x *FastHttpContext) Redirect(url string, statusCode int) {
	x.ctx.Redirect(url, statusCode)
}
func (x *FastHttpContext) CopyBodyAndStatusCode(resp *http.Response) {
	x.ctx.SetStatusCode(resp.StatusCode)
	x.ctx.SetBodyStream(resp.Body, -1)
}

func (x *FastHttpContext) Next() {
	if x.handlers == nil {
		return
	}

	if x.handlerIndex < x.handlerCount-1 {
		x.handlerIndex++
		x.handlers[x.handlerIndex](x)
	}
}
func (x *FastHttpContext) Reset() {
	x.ctx = nil
	x.sess = nil
	x.sessStore = nil
	x.cookieEncryptor = nil
	x.mapPool = nil
	x.handlers = nil
	x.handlerCount = 0
	x.handlerIndex = 0
}
