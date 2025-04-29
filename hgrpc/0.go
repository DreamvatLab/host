package hgrpc

import (
	"context"
	"fmt"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xconv"
	"github.com/DreamvatLab/go/xerr"

	_ "github.com/DreamvatLab/host/hconsul" // call init function in /hconsul/consul.go to register resolver
	oauth2go "github.com/DreamvatLab/oauth2go/core"
	"github.com/pascaldekloe/jwt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Define a custom type for context keys
// Fix warning: should not use built-in type string as key for value; define your own type to avoid collisions (SA1029)
type contextKey string

const (
	Header_Token = "token"
	// Ctx_Claims   = "claims"
	Ctx_Claims contextKey = "claims"
)

// // NewClientWithToken creates a gRPC client connection with token authentication
// func NewClientWithToken(addr string, ctx host.IHttpContext) (r *grpc.ClientConn, err error) {
// 	j := ctx.GetItem(host.Ctx_Token) // RL00002
// 	if j != nil {
// 		token, ok := j.(string)
// 		if ok {
// 			r, err = grpc.NewClient(
// 				addr,
// 				grpc.WithTransportCredentials(insecure.NewCredentials()),
// 				grpc.WithPerRPCCredentials(newTokenCredential(token, false)),
// 			)
// 		}
// 	}

// 	if r == nil {
// 		r, err = grpc.NewClient(
// 			addr,
// 			grpc.WithTransportCredentials(insecure.NewCredentials()),
// 		)
// 	}

// 	return r, xerr.WithStack(err)
// }

// receiveTokenMiddleware token receiving middleware
func receiveTokenMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	claims, err := receiveTokenMiddleware_ExtractClaims(ctx) // Extract claims from received token
	if err != nil || claims == nil {
		xerr.LogError(err)
		return handler(ctx, req)
	}

	// Successfully extracted, attach to context
	return handler(context.WithValue(ctx, Ctx_Claims, claims), req) // RL00003
}

func receiveTokenMiddleware_ExtractClaims(ctx context.Context) (*map[string]interface{}, error) {
	if metas, ok := metadata.FromIncomingContext(ctx); ok {
		if tokenArray, ok := metas[Header_Token]; ok {
			if len(tokenArray) == 1 {
				claims, err := jwt.ParseWithoutCheck(xbytes.StrToBytes(tokenArray[0]))
				if err != nil {
					return nil, xerr.WithStack(err)
				}

				claims.Set[oauth2go.Claim_Subject] = claims.Subject

				return &claims.Set, nil
			}
		}
	}
	return nil, nil
}

func getClaims(ctx context.Context) *map[string]interface{} {
	j, ok := ctx.Value(Ctx_Claims).(*map[string]interface{}) // RL00003

	if ok {
		return j
	}

	return nil
}

func getClaimValue(ctx context.Context, claimName string) interface{} {
	claims := getClaims(ctx)
	if claims != nil {
		if v, ok := (*claims)[claimName]; ok {
			return v
		}
	}
	return nil
}

func GetClaimString(ctx context.Context, claimName string) string {
	v := getClaimValue(ctx, claimName)
	return xconv.ToString(v)
}

func GetClaimInt64(ctx context.Context, claimName string) int64 {
	v := getClaimValue(ctx, claimName)
	return xconv.ToInt64(v)
}

type NewClientOptions struct {
	URL                string // URL of the gRPC server
	ConsulAddr         string // Address of the Consul server
	ServiceName        string // Name of the service to connect to
	ConsulToken        string // Token for the Consul server
	JwtToken           string // JWT token for authentication
	MaxCallRecvMsgSize int    // Maximum size of received messages
	MaxCallSendMsgSize int    // Maximum size of sent messages
}

// NewClient creates a gRPC client connection
func NewClient(options *NewClientOptions) (*grpc.ClientConn, error) {
	var url string
	if options.URL != "" {
		url = options.URL
	} else if options.ConsulAddr != "" && options.ServiceName != "" {
		url = fmt.Sprintf("%s://%s/%s", "consul", options.ConsulAddr, options.ServiceName)

		// if len(consulArgs) > 0 {
		// 	url = url + "?"
		// 	for k, v := range consulArgs {
		// 		url = url + k + "=" + v
		// 	}
		// }

		if options.ConsulToken != "" {
			url = url + "?token=" + options.ConsulToken
		}
	} else {
		return nil, xerr.New("url or consul settings must be provided")
	}

	if options.MaxCallRecvMsgSize <= 0 {
		options.MaxCallRecvMsgSize = 10 * 1024 * 1024
	}
	if options.MaxCallSendMsgSize <= 0 {
		options.MaxCallSendMsgSize = 10 * 1024 * 1024
	}

	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),               // Temporary solution, need to be replaced with a more secure transport credentials if needed
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`), // Temporary solution, need to be replaced with a more efficient load balancing policy if needed
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(options.MaxCallRecvMsgSize), // Set maximum size of received messages to 10MB
			grpc.MaxCallSendMsgSize(options.MaxCallSendMsgSize), // Set maximum size of sent messages to 10MB
		),
	}

	if options.JwtToken != "" {
		// Add JWT token authentication credentials for each RPC call
		// The second parameter false indicates this is not a streaming call
		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(newTokenCredential(options.JwtToken, false)))
	}

	r, err := grpc.NewClient(url, dialOptions...)
	if err != nil {
		return nil, xerr.WithStack(err)
	}

	return r, nil
}
