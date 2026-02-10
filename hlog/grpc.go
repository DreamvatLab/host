// Package hlog provides gRPC-based logging functionality
package hlog

import (
	"context"
	"fmt"

	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xtime"
	"github.com/DreamvatLab/host/hconsul"
	"github.com/DreamvatLab/host/hgrpc"
	"github.com/DreamvatLab/logs"
)

// grpcSink implements a logging sink that sends logs to a gRPC service
type grpcSink struct {
	clientID         string                     // Unique identifier for the client sending logs
	logServiceClient logs.LogEntryServiceClient // gRPC client for log service
}

// WriteLog writes a log entry to the gRPC service
func (o *grpcSink) WriteLog(entry *xlog.LogEntry) {
	_, err := o.logServiceClient.WriteLogEntry(context.Background(), &logs.WriteLogCommand{
		ClientID: o.clientID,
		LogEntry: &logs.LogEntry{
			Message:      entry.Message,
			StackTrace:   entry.Stack,
			Level:        logs.LogLevel(entry.Level),
			CreatedOnUtc: xtime.UTCNowUnixMS(),
		},
	})

	if err != nil {
		fmt.Println(err.Error())
	}
}

// NewGrpcSink creates a new gRPC logging sink
// consulConfig: Consul configuration
// clientID: Unique identifier for the logging client
//
// Returns: A new GrpcSink instance implementing the LogSink interface
func NewGrpcSink(consulConfig *hconsul.ConsulConfig, clientID string) xlog.LogSink {
	logServiceConn, err := hgrpc.NewClient(&hgrpc.NewClientOptions{
		ConsulAddr:  consulConfig.Addr,
		ConsulToken: consulConfig.Token,
		ServiceName: "logs",
	})
	xerr.FatalIfErr(err)

	logServiceClient := logs.NewLogEntryServiceClient(logServiceConn)

	return &grpcSink{
		logServiceClient: logServiceClient,
		clientID:         clientID,
	}
}
