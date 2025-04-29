package hgrpc

import (
	"net"

	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/host/hservice"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	panichandler "github.com/kazegusuri/grpc-panic-handler"
	"google.golang.org/grpc"
)

type GRPCOption func(*GRPCServiceHost)

type IGRPCServiceHost interface {
	hservice.IServiceHost
	GetGRPCServer() *grpc.Server
}

type GRPCServiceHost struct {
	hservice.ServiceHost
	GRPCServer     *grpc.Server
	MaxRecvMsgSize int
	MaxSendMsgSize int
}

func NewGRPCServiceHost(cp xconfig.IConfigProvider, options ...GRPCOption) IGRPCServiceHost {
	x := new(GRPCServiceHost)
	cp.GetStruct("@this", &x)
	x.ConfigProvider = cp

	for _, o := range options {
		o(x)
	}

	x.BuildGRPCServiceHost()

	return x
}

func (x *GRPCServiceHost) BuildGRPCServiceHost() {
	x.ServiceHost.BuildServiceHost()

	if x.MaxRecvMsgSize == 0 {
		x.MaxRecvMsgSize = 10 * 1024 * 1024
	}

	if x.MaxSendMsgSize == 0 {
		x.MaxSendMsgSize = 10 * 1024 * 1024
	}

	// GRPC Server
	unaryHandler := grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(panichandler.UnaryPanicHandler, receiveTokenMiddleware))
	streamHandler := grpc.StreamInterceptor(panichandler.StreamPanicHandler)
	panichandler.InstallPanicHandler(func(r interface{}) {
		xlog.Error(r)
	})

	x.GRPCServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(x.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(x.MaxSendMsgSize),
		unaryHandler,
		streamHandler,
	)
}

func (x *GRPCServiceHost) GetGRPCServer() *grpc.Server {
	return x.GRPCServer
}

func (x *GRPCServiceHost) Run() error {
	if x.ListenAddr == "" {
		xlog.Fatal("ListenAddr cannot be empty")
	}

	listen, err := net.Listen("tcp", x.ListenAddr)
	if err != nil {
		return xerr.WithStack(err)
	}

	xlog.Infof("Listening on %s", x.ListenAddr)
	return xerr.WithStack(x.GRPCServer.Serve(listen))
}
