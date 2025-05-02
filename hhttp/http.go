package hhttp

import (
	"bytes"
	"net/http"

	"github.com/DreamvatLab/go/xbytes"
	"github.com/DreamvatLab/go/xerr"
	"github.com/DreamvatLab/go/xlog"
	"github.com/DreamvatLab/go/xsync"
)

var (
	_bufferPool = xsync.NewSyncBufferPool(1024)
)

func GetRespBuffer(resp *http.Response, err error) (*bytes.Buffer, error) {
	if err != nil {
		return nil, err
	}

	bf := _bufferPool.GetBuffer()
	_, err = bf.ReadFrom(resp.Body)
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		return nil, xerr.WithStack(err)
	}

	if resp.StatusCode != http.StatusOK {
		xlog.Warnf("%s %s [%d] -> %s", resp.Request.Method, resp.Request.URL.String(), resp.StatusCode, xbytes.BytesToStr(bf.Bytes()))
	}

	return bf, nil
}

func RecycleBuffer(buffer *bytes.Buffer) {
	_bufferPool.PutBuffer(buffer)
}
