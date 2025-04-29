package hgrpc

import (
	"context"

	"google.golang.org/grpc/credentials"
)

type tokenCredential struct {
	Token      string
	RequireTLS bool
}

func newTokenCredential(token string, requireTLS bool) credentials.PerRPCCredentials {
	return &tokenCredential{
		Token:      token,
		RequireTLS: requireTLS,
	}
}

// GetRequestMetadata gets request metadata
func (x *tokenCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	if x.Token != "" {
		return map[string]string{
			// _authHeader: _tokenType + x.Token,
			Header_Token: x.Token,
		}, nil
	}

	return map[string]string{}, nil
}

// RequireTransportSecurity indicates whether TLS is required
func (x *tokenCredential) RequireTransportSecurity() bool {
	return x.RequireTLS
}
