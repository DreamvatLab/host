package hhttp

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/DreamvatLab/go/xhttp"
	"github.com/DreamvatLab/host/hurl"
)

type APIClient struct {
	URLProvider hurl.IURLProvider
}

func (x *APIClient) DoBuffer(client *http.Client, method, url string, configRequest func(*http.Request), bodyObj interface{}) (buffer *bytes.Buffer, err error) {
	buffer = _bufferPool.GetBuffer()

	var resp *http.Response
	resp, err = x.Do(client, method, url, configRequest, bodyObj)
	if err != nil {
		return nil, err
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	// 读取Response Body
	_, err = buffer.ReadFrom(resp.Body)
	if err != nil {
		return nil, err
	}

	return buffer, err
}

func (x *APIClient) Do(client *http.Client, method, url string, configRequest func(*http.Request), bodyObj interface{}) (resp *http.Response, err error) {
	var request *http.Request

	if x.URLProvider != nil {
		// 渲染Url
		url = x.URLProvider.RenderURLCache(url)
	}

	// 创建Request
	bodyBuffer := _bufferPool.GetBuffer()
	if bodyObj != nil {
		switch v := bodyObj.(type) {
		case []byte:
			bodyBuffer.Write(v)
		case string:
			bodyBuffer.WriteString(v)
		default:
			var body []byte
			body, err = json.Marshal(v)
			if err != nil {
				return nil, err
			}
			bodyBuffer.Write(body)
		}

		request, err = http.NewRequest(method, url, bodyBuffer)
	} else {
		request, err = http.NewRequest(method, url, nil)
	}
	defer func() { _bufferPool.PutBuffer(bodyBuffer) }()

	if err != nil {
		return nil, err
	}

	// 配置Request
	request.Header.Set(xhttp.HEADER_CTYPE, xhttp.CTYPE_JSON)
	if configRequest != nil {
		configRequest(request)
	}

	// 发送请求
	resp, err = client.Do(request)
	if err != nil {
		return nil, err
	}

	return resp, err
}
