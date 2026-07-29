package hclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xutils"
	"github.com/DreamvatLab/host"
	oauth2core "github.com/DreamvatLab/oauth2go/core"
	"github.com/pascaldekloe/jwt"
	"golang.org/x/oauth2"
)

type OAuthClientHandler struct {
	OAuth              *host.OAuthOptions
	ContextTokenStore  host.IContextTokenStore
	UserJsonSessionkey string
	UserIDSessionKey   string
	TokenCookieName    string
}

func NewOAuthClientHandler(
	oauthOptions *host.OAuthOptions,
	contextTokenStore host.IContextTokenStore,
	userJsonSessionkey string,
	userIDSessionKey string,
	tokenCookieName string,
) host.IOAuthClientHandler {
	return &OAuthClientHandler{
		OAuth:              oauthOptions,
		ContextTokenStore:  contextTokenStore,
		UserJsonSessionkey: userJsonSessionkey,
		UserIDSessionKey:   userIDSessionKey,
		TokenCookieName:    tokenCookieName,
	}
}

func (x *OAuthClientHandler) SignInHandler(ctx host.IHttpContext) {
	returnURL := ctx.GetFormString(oauth2core.Form_ReturnUrl)
	if returnURL == "" {
		returnURL = "/"
	}

	userStr := ctx.GetSessionString(x.UserJsonSessionkey)
	if userStr != "" {
		// Already logged in
		ctx.Redirect(returnURL, http.StatusFound)
		return
	}

	// Record request URL and redirect to login page
	host.RedirectAuthorizeEndpoint(ctx, x.OAuth, returnURL)
}

func (x *OAuthClientHandler) SignInCallbackHandler(ctx host.IHttpContext) {
	state := ctx.GetFormString(oauth2core.Form_State)
	redirectUrl := ctx.GetSessionString(state)
	if redirectUrl == "" {
		ctx.WriteString("invalid state")
		ctx.SetStatusCode(http.StatusBadRequest)
		return
	}
	ctx.RemoveSession(state) // Free memory

	// PKCE：RFC 6749 §4.1.2 授权回调只回传 code/state，不再回显 code_challenge。
	// 客户端只需从 session 取出 code_verifier，在 token 交换时提交；由授权服务器校验。
	var sessionCodeVerifier string
	if x.OAuth.PkceRequired {
		sessionCodeVerifier = ctx.GetSessionString(oauth2core.Form_CodeVerifier)
		if sessionCodeVerifier == "" {
			ctx.WriteString("pkce code verifier does not exist in store")
			ctx.SetStatusCode(http.StatusBadRequest)
			return
		}
		ctx.RemoveSession(oauth2core.Form_CodeVerifier)
		ctx.RemoveSession(oauth2core.Form_CodeChallengeMethod)
	}

	// Exchange token
	code := ctx.GetFormString(oauth2core.Form_Code)
	httpCtx := context.Background()
	var oauth2Token *oauth2.Token
	var err error

	// Get old refresh token and send it to Auth server for logout
	token, _ := x.ContextTokenStore.GetToken(ctx)
	var refreshTokenOption oauth2.AuthCodeOption
	if token != nil && token.RefreshToken != "" {
		refreshTokenOption = oauth2.SetAuthURLParam(oauth2core.Form_RefreshToken, token.RefreshToken)
	}

	if x.OAuth.PkceRequired {
		// RFC 7636：token 请求只需 code_verifier
		codeVerifierParam := oauth2.SetAuthURLParam(oauth2core.Form_CodeVerifier, sessionCodeVerifier)

		if refreshTokenOption != nil {
			oauth2Token, err = x.OAuth.Exchange(httpCtx, code, codeVerifierParam, refreshTokenOption)
		} else {
			oauth2Token, err = x.OAuth.Exchange(httpCtx, code, codeVerifierParam)
		}
	} else {
		if refreshTokenOption != nil {
			oauth2Token, err = x.OAuth.Exchange(httpCtx, code, refreshTokenOption)
		} else {
			oauth2Token, err = x.OAuth.Exchange(httpCtx, code)
		}
	}

	if xerr.LogError(err) {
		ctx.WriteString(err.Error())
		ctx.SetStatusCode(http.StatusInternalServerError)
		return
	}

	// Convert string to token object
	jwtToken, err := jwt.ParseWithoutCheck(xbytes.StrToBytes(oauth2Token.AccessToken))
	if err == nil {
		userStr := xbytes.BytesToStr(jwtToken.Raw)
		ctx.SetSession(x.UserJsonSessionkey, userStr)
		if jwtToken.Subject != "" {
			ctx.SetSession(x.UserIDSessionKey, jwtToken.Subject)
		}

		// Save token
		x.ContextTokenStore.SaveToken(ctx, oauth2Token)

		// Redirect to pre-login page
		ctx.Redirect(redirectUrl, http.StatusFound)
	} else {
		ctx.WriteString(err.Error())
		xerr.LogError(err)
	}
}

func (x *OAuthClientHandler) SignOutHandler(ctx host.IHttpContext) {
	// Go to Passport for logout
	state := xutils.RandomString(32)
	returnUrl := ctx.GetFormString(oauth2core.Form_ReturnUrl)
	if returnUrl == "" {
		returnUrl = "/"
	}
	ctx.SetSession(state, returnUrl)
	targetURL := fmt.Sprintf("%s?%s=%s&%s=%s&%s=%s",
		x.OAuth.EndSessionEndpoint,
		oauth2core.Form_ClientID,
		x.OAuth.ClientID,
		oauth2core.Form_RedirectUri,
		url.PathEscape(x.OAuth.SignOutRedirectURL),
		oauth2core.Form_State,
		url.QueryEscape(state),
	)
	ctx.Redirect(targetURL, http.StatusFound)
}

func (x *OAuthClientHandler) SignOutCallbackHandler(ctx host.IHttpContext) {
	state := ctx.GetFormString(oauth2core.Form_State)
	returnURL := ctx.GetSessionString(state)
	if returnURL == "" {
		ctx.WriteString("invalid state")
		ctx.SetStatusCode(http.StatusBadRequest)
		return
	}

	endSessionID := ctx.GetFormString(oauth2core.Form_EndSessionID)
	if endSessionID == "" {
		ctx.WriteString("missing es_id")
		ctx.SetStatusCode(http.StatusBadRequest)
		return
	}

	token, _ := x.ContextTokenStore.GetToken(ctx)
	if token != nil {
		// Request Auth server to delete old RefreshToken
		data := make(url.Values, 5)
		data[oauth2core.Form_State] = []string{state}
		data[oauth2core.Form_EndSessionID] = []string{endSessionID}
		data[oauth2core.Form_ClientID] = []string{x.OAuth.ClientID}
		data[oauth2core.Form_ClientSecret] = []string{x.OAuth.ClientSecret}
		data[oauth2core.Form_RefreshToken] = []string{token.RefreshToken}
		http.PostForm(x.OAuth.EndSessionEndpoint, data)
	}

	host.SignOut(ctx, x.TokenCookieName)

	// Redirect back to the page before logout
	ctx.Redirect(returnURL, http.StatusFound)
}
