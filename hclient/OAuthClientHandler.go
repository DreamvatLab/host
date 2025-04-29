package hclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
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

	var sessionCodeVerifier, sessionSodeChallengeMethod string
	if x.OAuth.PkceRequired {
		sessionCodeVerifier = ctx.GetSessionString(oauth2core.Form_CodeVerifier)
		if sessionCodeVerifier == "" {
			ctx.WriteString("pkce code verifier does not exist in store")
			ctx.SetStatusCode(http.StatusBadRequest)
			return
		}
		ctx.RemoveSession(oauth2core.Form_CodeVerifier)
		sessionSodeChallengeMethod = ctx.GetSessionString(oauth2core.Form_CodeChallengeMethod)
		if sessionSodeChallengeMethod == "" {
			ctx.WriteString("pkce transformation method does not exist in store")
			ctx.SetStatusCode(http.StatusBadRequest)
			return
		}
		ctx.RemoveSession(oauth2core.Form_CodeChallengeMethod)

		codeChallenge := ctx.GetFormString(oauth2core.Form_CodeChallenge)
		codeChallengeMethod := ctx.GetFormString(oauth2core.Form_CodeChallengeMethod)

		if sessionSodeChallengeMethod != codeChallengeMethod {
			ctx.WriteString("pkce transformation method does not match")
			xlog.Debugf("session method: '%s', incoming method:'%s'", sessionSodeChallengeMethod, codeChallengeMethod)
			ctx.SetStatusCode(http.StatusBadRequest)
			return
		} else if (sessionSodeChallengeMethod == oauth2core.Pkce_Plain && codeChallenge != oauth2core.ToSHA256Base64URL(sessionCodeVerifier)) ||
			(sessionSodeChallengeMethod == oauth2core.Pkce_Plain && codeChallenge != sessionCodeVerifier) {
			ctx.WriteString("pkce code verifiver and chanllenge does not match")
			xlog.Debugf("session verifiver: '%s', incoming chanllenge:'%s'", sessionCodeVerifier, codeChallenge)
			ctx.SetStatusCode(http.StatusBadRequest)
			return
		}
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
		codeChanllengeParam := oauth2.SetAuthURLParam(oauth2core.Form_CodeVerifier, sessionCodeVerifier)
		codeChanllengeMethodParam := oauth2.SetAuthURLParam(oauth2core.Form_CodeChallengeMethod, sessionSodeChallengeMethod)

		// Send token exchange request
		if refreshTokenOption != nil {
			oauth2Token, err = x.OAuth.Exchange(httpCtx, code, codeChanllengeParam, codeChanllengeMethodParam, refreshTokenOption)
		} else {
			oauth2Token, err = x.OAuth.Exchange(httpCtx, code, codeChanllengeParam, codeChanllengeMethodParam)
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
