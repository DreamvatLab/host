// Package hlog provides gRPC-based logging functionality

package hlog

import (
	"testing"
	"time"

	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/host/hconsul"
)

func Test_grpcSink_WriteLog(t *testing.T) {
	// Setup test sink
	sink := NewGrpcSink(&hconsul.ConsulConfig{
		Addr:  "localhost:8500",
		Token: "48359c9b-afc2-341f-9a61-ee37f3b361b6",
	}, "SOLAR")

	// Test case 1: Normal log writing
	t.Run("Normal log writing", func(t *testing.T) {
		sink.WriteLog(&xlog.LogEntry{
			Level:   0, // Info level
			Time:    time.Now(),
			Message: "This is a test log message",
		})
	})

	// Test case 2: Formatted string log
	t.Run("Formatted string log", func(t *testing.T) {
		sink.WriteLog(&xlog.LogEntry{
			Level:   -1, // Debug level
			Time:    time.Now(),
			Message: "User john performed action login",
		})
	})

	// Test case 3: Direct value log
	t.Run("Direct value log", func(t *testing.T) {
		sink.WriteLog(&xlog.LogEntry{
			Level:   1, // Warn level
			Time:    time.Now(),
			Message: "42 is the answer",
		})
	})

	// Test case 4: Different log levels
	t.Run("Different log levels", func(t *testing.T) {
		levels := []int{-1, 0, 1, 2, 3} // Debug, Info, Warn, Error, Fatal
		for _, level := range levels {
			sink.WriteLog(&xlog.LogEntry{
				Level:   level,
				Time:    time.Now(),
				Message: "Testing log level",
			})
		}
	})
}
