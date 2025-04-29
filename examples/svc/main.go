package main

import (
	"context"
	"shared"

	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/host/hconsul"
	"github.com/DreamvatLab/host/hgrpc"
)

func main() {
	cp := xconfig.NewJsonConfigProvider()

	hconsul.RegisterServiceInfo(cp)

	host := hgrpc.NewGRPCServiceHost(cp)

	shared.RegisterTestServiceServer(host.GetGRPCServer(), &TestService{})

	xerr.FatalIfErr(host.Run())
}

type TestService struct {
	shared.UnimplementedTestServiceServer
}

func (s *TestService) Test(ctx context.Context, req *shared.TestRequest) (*shared.TestResponse, error) {
	return &shared.TestResponse{Message: "Hello, " + req.Name}, nil
}
