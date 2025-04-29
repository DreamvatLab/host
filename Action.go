package host

import (
	"strings"

	"github.com/DreamvatLab/go/xlog"
)

type ActionGroup struct {
	PreHandlers   []RequestHandler
	Actions       []*Action
	AfterHandlers []RequestHandler
}

type Action struct {
	RouteKey   string
	Route      string
	Area       string
	Controller string
	Action     string
	Handlers   []RequestHandler
}

func NewActionGroup(preHandlers []RequestHandler, actions []*Action, afterHandlers ...RequestHandler) *ActionGroup {
	return &ActionGroup{
		PreHandlers:   preHandlers,
		Actions:       actions,
		AfterHandlers: afterHandlers,
	}
}

func NewAction(route, routeKey string, handlers ...RequestHandler) *Action {
	if len(handlers) == 0 {
		xlog.Fatal("handlers are missing")
	}

	var area, controller, action string
	routeArray := strings.Split("_", routeKey)
	if len(routeArray) == 3 {
		area = routeArray[0]
		controller = routeArray[1]
		action = routeArray[2]
	} else if len(routeArray) == 2 {
		area = routeArray[0]
		controller = routeArray[1]
	} else {
		area = routeArray[0]
	}

	return &Action{
		RouteKey:   routeKey,
		Route:      route,
		Area:       area,
		Action:     action,
		Controller: controller,
		Handlers:   handlers,
	}
}

func (x *Action) AppendHandler(handlers ...RequestHandler) {
	x.Handlers = append(x.Handlers, handlers...)
}
