package host

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xconv"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/go/xutils"
	"github.com/DreamvatLab/host/hmodel"
	oauth2core "github.com/DreamvatLab/oauth2go/core"
	"golang.org/x/oauth2"
)

func ConfigHttpClient(configProvider xconfig.IConfigProvider) {
	// HTTP client configuration
	skipCertVerification := configProvider.GetBool("Http.SkipCertVerification")
	xlog.Debugf("Skip http.DefaultClient certificate verification: %v", skipCertVerification)
	proxy := configProvider.GetString("Http.Proxy")
	if skipCertVerification || proxy != "" {
		// Use custom transport layer if any condition is met
		transport := new(http.Transport)
		if skipCertVerification {
			// Skip certificate verification
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: skipCertVerification}
		}
		if proxy != "" {
			// Use proxy
			transport.Proxy = func(r *http.Request) (*url.URL, error) {
				return url.Parse(proxy)
			}
		}
		http.DefaultClient.Transport = transport
	}
}

func GetUser(ctx IHttpContext, userJsonSessionkey string) (r *hmodel.User) {
	userJson := ctx.GetSessionString(userJsonSessionkey)
	if userJson != "" {
		// Already logged in
		err := json.Unmarshal(xbytes.StrToBytes(userJson), &r)
		xerr.LogError(err)
	}
	return
}

func GetRoutesByKey(routeKey string) (area, controller, action string) {
	routeArray := strings.Split(routeKey, Seperator_Route)

	if len(routeArray) >= 3 {
		action = routeArray[2]
	}

	if len(routeArray) >= 2 {
		controller = routeArray[1]
	}

	area = routeArray[0]

	return
}

func GetUserID(ctx IHttpContext, userIDSessionkey string) string {
	return ctx.GetSessionString(userIDSessionkey)
}

func SignOut(ctx IHttpContext, tokenCookieName string) {
	ctx.EndSession()
	ctx.RemoveCookie(tokenCookieName)
}

func RedirectAuthorizeEndpoint(ctx IHttpContext, oauthOptions *OAuthOptions, returnURL string) {
	state := xutils.RandomString(32)
	ctx.SetSession(state, returnURL)
	if oauthOptions.PkceRequired {
		codeVerifier := oauth2core.Random64String()
		codeChanllenge := oauth2core.ToSHA256Base64URL(codeVerifier)
		ctx.SetSession(oauth2core.Form_CodeVerifier, codeVerifier)
		ctx.SetSession(oauth2core.Form_CodeChallengeMethod, oauth2core.Pkce_S256)
		codeChanllengeParam := oauth2.SetAuthURLParam(oauth2core.Form_CodeChallenge, codeChanllenge)
		codeChanllengeMethodParam := oauth2.SetAuthURLParam(oauth2core.Form_CodeChallengeMethod, oauth2core.Pkce_S256)
		ctx.Redirect(oauthOptions.AuthCodeURL(state, codeChanllengeParam, codeChanllengeMethodParam), http.StatusFound)
	} else {
		ctx.Redirect(oauthOptions.AuthCodeURL(state), http.StatusFound)
	}
}

// func GenerateID() string {
// 	return _idGenerator.GenerateString()
// }

func HandleErr(err error, ctx IHttpContext) bool {
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		errID := xutils.GenerateStringID()
		xlog.Errorf("[%s] %+v", errID, err)
		ctx.WriteJsonBytes(xbytes.StrToBytes(`{"err":"` + errID + `"}`))

		return true
	}
	return false
}

func getClaims(ctx IHttpContext) *map[string]interface{} {
	j, ok := ctx.GetItem(Ctx_Claims).(*map[string]interface{}) // RL00001

	if ok {
		return j
	}

	return nil
}

func getClaimValue(ctx IHttpContext, claimName string) interface{} {
	claims := getClaims(ctx)
	if claims != nil {
		if v, ok := (*claims)[claimName]; ok {
			return v
		}
	}
	return nil
}

func GetClaimString(ctx IHttpContext, claimName string) string {
	v := getClaimValue(ctx, claimName)
	return xconv.ToString(v)
}

func GetClaimInt64(ctx IHttpContext, claimName string) int64 {
	v := getClaimValue(ctx, claimName)
	return xconv.ToInt64(v)
}

func GetEncryptedCookie(ctx IHttpContext, cookieEncryptor xsecurity.ICookieEncryptor, name string) string {
	encryptedCookie := ctx.GetCookieString(name)
	if encryptedCookie == "" {
		return ""
	}

	var r string
	err := cookieEncryptor.Decrypt(name, encryptedCookie, &r)

	if xerr.LogError(err) {
		return ""
	}

	return r
}
func SetEncryptedCookie(ctx IHttpContext, cookieEncryptor xsecurity.ICookieEncryptor, key, value string, options ...func(*http.Cookie)) {
	if encryptedCookie, err := cookieEncryptor.Encrypt(key, value); err == nil {
		ctx.SetCookieKV(key, encryptedCookie, options...)
	} else {
		xerr.LogError(err)
	}
}

func FlattenMap(input map[string][]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		if len(value) > 0 {
			output[key] = value[0]
		}
	}
	return output
}
