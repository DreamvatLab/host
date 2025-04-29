package hfasthttp

import "github.com/DreamvatLab/go/xlog"

type debugLogger struct{}

func (o *debugLogger) Printf(format string, args ...interface{}) {
	xlog.Debugf(format, args...)
}
