package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	en "url-collector/env"
	hh "url-collector/handlers"
)

func main() {
	port := en.GetEnvVar("PORT", "8080")

	l := log.New(os.Stdout, "url-collector.com", log.LstdFlags)
	server := hh.NewPictureUrlServer()

	s := &http.Server{
		Addr:         ":" + port,
		Handler:      server,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	gracefulShutdown(s, l)

}

func gracefulShutdown(s *http.Server, l *log.Logger) {
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			l.Fatal(err)
		}
	}()

	sigChan := make(chan os.Signal, 5)
	signal.Notify(sigChan, os.Interrupt)
	signal.Notify(sigChan, syscall.SIGTERM)
	sig := <-sigChan
	l.Println("Received terminate shutdown", sig)

	tc, cncl := context.WithTimeout(context.Background(), 30*time.Second)
	cncl()
	err := s.Shutdown(tc)
	if err != nil {
		log.Println(err)
	}
}
