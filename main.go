package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	cfg, cfgErr := LoadConfig()
	app := NewApp(cfg, cfgErr)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.routesManager(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       90 * time.Second,
	}

	log.Printf("Via Verde Crédito Rural %s iniciado na porta %s", AppVersion, cfg.Port)
	if cfgErr != nil {
		log.Printf("ATENÇÃO: %v", cfgErr)
	}
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("servidor finalizado: %v", err)
		os.Exit(1)
	}
}
