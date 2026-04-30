package come

func GenRouter(proj *Project) string {
	return `package server

import(
	"net/http"
)

func Chain(h http.Handler,middlewares ...func(http.Handler)http.Handler)http.Handler{
	for i:=len(middlewares)-1;i>=0;i--{
		h=middlewares[i](h)
	}
	return h
}
`
}
