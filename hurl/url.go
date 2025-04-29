package hurl

type IURLProvider interface {
	GetURL(urlKey string) string
	GetURLCache(urlKey string) string
	RenderURL(url string) string
	RenderURLCache(url string) string
}
