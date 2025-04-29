package hclient

import (
	"encoding/json"
	"net/http"

	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xsecurity"
	"github.com/DreamvatLab/host"
	"golang.org/x/oauth2"
)

const _cookieTokenProtectorKey = "token"

type CookieTokenStore struct {
	CookieEncryptor xsecurity.ICookieEncryptor
	TokenCookieName string
}

func NewCookieTokenStore(tokenCookieName string, cookieEncryptor xsecurity.ICookieEncryptor) *CookieTokenStore {
	return &CookieTokenStore{
		TokenCookieName: tokenCookieName,
		CookieEncryptor: cookieEncryptor,
	}
}

// SaveToken saves the token
func (x *CookieTokenStore) SaveToken(ctx host.IHttpContext, token *oauth2.Token) error {
	tokenJson, err := json.Marshal(token)
	if err != nil {
		return xerr.WithStack(err)
	}

	// Encrypt token
	securedString, err := x.CookieEncryptor.Encrypt(_cookieTokenProtectorKey, tokenJson)
	if err != nil {
		return err
	}

	ctx.SetCookieKV(x.TokenCookieName, securedString, func(c *http.Cookie) {
		c.HttpOnly = true
	})
	return nil
}

// GetToken gets the token
func (x *CookieTokenStore) GetToken(ctx host.IHttpContext) (*oauth2.Token, error) {
	// Get token from Session
	tokenJson := ctx.GetCookieString(x.TokenCookieName)
	if tokenJson == "" {
		return nil, nil
	}
	var tokenJsonBytes []byte
	err := x.CookieEncryptor.Decrypt(_cookieTokenProtectorKey, tokenJson, &tokenJsonBytes)
	if err != nil {
		return nil, xerr.WithStack(err)
	}

	t := new(oauth2.Token)
	err = json.Unmarshal(tokenJsonBytes, t)
	if err != nil {
		return nil, xerr.WithStack(err)
	}

	return t, xerr.WithStack(err)
}
