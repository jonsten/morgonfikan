package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jonsten/morgonfikan"
	"github.com/jonsten/morgonfikan/internal/httpapi"
	_ "time/tzdata"
)

func main() {
	config, err := morgonfikan.LoadEmbeddedConfig()
	if err != nil {
		log.Fatalf("load menu configuration: %v", err)
	}
	zone, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		log.Fatalf("load Stockholm time zone: %v", err)
	}

	server := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.New(config, zone).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shut down HTTP server: %v", err)
		}
	}()

	log.Printf("listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", err)
	}
}
