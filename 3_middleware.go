package host

import "github.com/DreamvatLab/go/xhttp"

func JsonConentTypeHandler(ctx IHttpContext) {
	ctx.SetContentType(xhttp.CTYPE_JSON)
	ctx.Next()
}
