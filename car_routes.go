package main

import "net/http"

func (a *App) routesManagerCAR() http.Handler {
	a.installAutoProjectTemplate()
	base := a.routesManagerReliable()
	mux := http.NewServeMux()
	mux.Handle("GET /health", securityHeaders(recoverer(logger(http.HandlerFunc(a.healthDB)))))
	mux.Handle("GET /car", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.carPage))))))
	mux.Handle("GET /car/kml", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.carKML))))))
	mux.Handle("GET /car/demonstrativo", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.carDemonstrativoPDF))))))
	mux.Handle("GET /car/demonstrativo.json", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.carDemonstrativoJSON))))))

	mux.Handle("GET /autoproject", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.autoProjectPage))))))
	mux.Handle("POST /autoproject/analyze", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.autoProjectAnalyzeV2))))))

	mux.Handle("GET /clients", securityHeaders(recoverer(logger(a.withAuth(http.HandlerFunc(a.managerClientsList))))))
	mux.Handle("POST /clients", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.managerClientCreate))))))
	mux.Handle("GET /clients/{id}", securityHeaders(recoverer(logger(a.withAuth(http.HandlerFunc(a.managerClientDetail))))))
	mux.Handle("POST /clients/{id}/edit", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.managerClientEdit))))))
	mux.Handle("POST /clients/{id}/archive", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.managerClientArchiveToggle))))))

	mux.Handle("GET /projects/new", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.managerProjectNewGet))))))
	mux.Handle("POST /projects/new", securityHeaders(recoverer(logger(a.ownerOnly(http.HandlerFunc(a.managerProjectNewPost))))))

	mux.Handle("/", base)
	return mux
}
