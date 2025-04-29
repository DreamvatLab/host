package main

import (
	"context"
	"shared"

	"github.com/DreamvatLab/go/xconfig"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/host"
	"github.com/DreamvatLab/host/hconsul"
	"github.com/DreamvatLab/host/hfasthttp"
	"github.com/DreamvatLab/host/hgrpc"
)

func main() {
	cp := xconfig.NewJsonConfigProvider()

	server := hfasthttp.NewFHOAuthResourceHost(cp)

	consulConfig, err := hconsul.GetConsulConfig(cp)
	xerr.FatalIfErr(err)

	gprcConn, err := hgrpc.NewClient(&hgrpc.NewClientOptions{
		ConsulAddr:  consulConfig.Addr,
		ConsulToken: consulConfig.Token,
		ServiceName: "img",
	})
	xerr.FatalIfErr(err)

	testServiceClient := shared.NewTestServiceClient(gprcConn)

	server.AddAction("GET/test", "__test", func(ctx host.IHttpContext) {
		resp, err := testServiceClient.Test(context.Background(), &shared.TestRequest{Name: "John"})
		if host.HandleErr(err, ctx) {
			return
		}
		ctx.WriteString(resp.Message)
	})

	xerr.FatalIfErr(server.Run())
}
