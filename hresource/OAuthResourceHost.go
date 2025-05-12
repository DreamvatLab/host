package hresource

import (
	"crypto/rsa"
	"net/http"
	"strings"
	"time"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xconv"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xhttp"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsecurity/xrsa"
	"github.com/DreamvatLab/go/xslice"
	"github.com/DreamvatLab/host"
	oauth2core "github.com/DreamvatLab/oauth2go/core"
	"github.com/DreamvatLab/oauth2go/model"
	"github.com/pascaldekloe/jwt"
)

type IOAuthResourceHost interface {
	host.IBaseHost
	host.IWebHost
	AuthHandler(ctx host.IHttpContext)
}

type OAuthResourceHost struct {
	host.BaseHost
	OAuthOptions     *model.Resource `json:"OAuth,omitempty"`
	PublicKey        *rsa.PublicKey
	PublicKeyPath    string
	SigningAlgorithm string
	// TokenValidator   func(*jwt.Claims) string
}

func (x *OAuthResourceHost) BuildOAuthResourceHost() {
	x.BaseHost.BuildBaseHost()

	if x.OAuthOptions == nil {
		xlog.Fatal("OAuth secion in configuration is missing")
	}

	if x.PublicKeyPath == "" {
		xlog.Fatal("public key path cannot be empty")
	}
	if x.OAuthOptions == nil {
		xlog.Fatal("oauth options cannot be nil")
	}
	if len(x.OAuthOptions.ValidIssuers) == 0 {
		xlog.Fatal("Issuers cannot be empty")
	}
	if len(x.OAuthOptions.ValidAudiences) == 0 {
		xlog.Fatal("Audiences cannot be empty")
	}
	if x.SigningAlgorithm == "" {
		x.SigningAlgorithm = jwt.PS256
	}

	if x.URLProvider != nil {
		for i := range x.OAuthOptions.ValidIssuers {
			x.OAuthOptions.ValidIssuers[i] = x.URLProvider.RenderURL(x.OAuthOptions.ValidIssuers[i])
		}
		for i := range x.OAuthOptions.ValidAudiences {
			x.OAuthOptions.ValidAudiences[i] = x.URLProvider.RenderURL(x.OAuthOptions.ValidAudiences[i])
		}
	}

	// read public certificate
	cert, err := xrsa.ReadCertFromFile(x.PublicKeyPath)
	xerr.FatalIfErr(err)
	x.PublicKey = cert.PublicKey.(*rsa.PublicKey)
}

func (x *OAuthResourceHost) AuthHandler(ctx host.IHttpContext) {
	routeKey := ctx.GetItemString(host.Ctx_RouteKey)
	area, controller, action := host.GetRoutesByKey(routeKey)

	authHeader := ctx.GetHeader(xhttp.HEADER_AUTH)
	if authHeader == "" {
		if x.PermissionAuditor.CheckRouteWithLevel(area, controller, action, 0, 0, []string{}) {
			ctx.Next() // 没有提供令牌，但是允许匿名访问
			return
		} else {
			// 没有提供令牌，且不允许匿名访问
			ctx.SetStatusCode(http.StatusUnauthorized)
			ctx.WriteString("Authorization header is missing")
			return
		}
	}

	// verify authorization header
	array := strings.Split(authHeader, " ")
	if len(array) != 2 || array[0] != host.AuthType_Bearer {
		ctx.SetStatusCode(http.StatusBadRequest)
		xlog.Warnf("'%s'invalid authorization header format. '%s'", ctx.GetRemoteIP(), authHeader)
		return
	}
	token := array[1]

	// verify signature
	jwtClaims, err := jwt.RSACheck(xbytes.StrToBytes(token), x.PublicKey)
	if err != nil {
		ctx.SetStatusCode(http.StatusUnauthorized)
		xlog.Warn("'"+ctx.GetRemoteIP()+"'", err)
		return
	}

	// validate time limits
	isNotExpired := jwtClaims.Valid(time.Now().UTC())
	if !isNotExpired {
		ctx.SetStatusCode(http.StatusUnauthorized)
		msgCode := "current time not in token's valid period"
		ctx.WriteString(msgCode)
		xlog.Warnf("%s. Remote IP:[%s]", msgCode, ctx.GetRemoteIP())
		return
	}

	// validate aud
	isValidAudience := x.OAuthOptions.ValidAudiences != nil && xslice.HasAnyStr(x.OAuthOptions.ValidAudiences, jwtClaims.Audiences)
	if !isValidAudience {
		ctx.SetStatusCode(http.StatusUnauthorized)
		msgCode := "invalid audience"
		ctx.WriteString(msgCode)
		xlog.Warnf("%s. Required: %v, has: %v, IP:[%s]", msgCode, x.OAuthOptions.ValidAudiences, jwtClaims.Audiences, ctx.GetRemoteIP())
		return
	}

	// validate iss
	isValidIssuer := x.OAuthOptions.ValidIssuers != nil && xslice.HasStr(x.OAuthOptions.ValidIssuers, jwtClaims.Issuer)
	if !isValidIssuer {
		ctx.SetStatusCode(http.StatusUnauthorized)
		msgCode := "invalid issuer"
		ctx.WriteString(msgCode)
		xlog.Warnf("%s. Required: %v, has: %v, IP:[%s]", msgCode, x.OAuthOptions.ValidIssuers, jwtClaims.Issuer, ctx.GetRemoteIP())
		return
	}

	// if x.TokenValidator != nil {
	// 	if msgCode := x.TokenValidator(token); msgCode != "" {
	// 		ctx.SetStatusCode(http.StatusUnauthorized)
	// 		ctx.WriteString(msgCode)
	// 		xlog.Warn("'"+ctx.GetRemoteIP()+"'", msgCode)
	// 		return
	// 	}
	// }

	var msgCode string
	if jwtClaims != nil {

		roles := xconv.ToInt64(jwtClaims.Set[oauth2core.Claim_Role])
		level := xconv.ToInt64(jwtClaims.Set[oauth2core.Claim_Level])

		var userScopes []string
		if rawScopes, ok := jwtClaims.Set[oauth2core.Claim_Scope]; ok {
			switch v := rawScopes.(type) {
			case []string:
				userScopes = v
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok {
						userScopes = append(userScopes, str)
					}
				}
			default:
				userScopes = []string{}
			}
		} else {
			userScopes = []string{}
		}

		if x.PermissionAuditor.CheckRouteWithLevel(area, controller, action, roles, int32(level), userScopes) {
			// Has permission, allow
			ctx.SetItem(host.Ctx_UserID, jwtClaims.Subject) // UserID
			ctx.SetItem(host.Ctx_Claims, &jwtClaims.Set)    // RL00001
			ctx.SetItem(host.Ctx_Token, token)              // RL00002
			ctx.Next()
			return
		} else {
			msgCode = "permission denied"
		}
	}

	// Not allow
	ctx.SetStatusCode(http.StatusUnauthorized)
	ctx.WriteString(msgCode)
}
