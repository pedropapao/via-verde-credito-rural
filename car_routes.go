package main

import "net/http"

func (a *App) routesManagerCAR() http.Handler {
	base := a.routesManagerReliable()
	mux := http.NewServeMux()
	mux.Handle("GET /car", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.carPage))))))
	mux.Handle("GET /car/kml", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.carKML))))))
	mux.Handle("/", base)
	return mux
}
