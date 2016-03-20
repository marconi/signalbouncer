package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/marconi/signalbouncer"
)

var (
	address = flag.String("address", "127.0.0.1:8080", "Set listening address")
	tlscert = flag.String("tlscert", "", "TLS certificate path")
	tlskey  = flag.String("tlskey", "", "TLS key path")
)

func main() {
	flag.Parse()

	signalRooms := signalbouncer.NewSignalRooms()
	signalService := signalbouncer.NewSignalService()
	handler := signalbouncer.NewHandler(signalRooms, signalService)

	go func() {
		log.Infof("listening on %s", *address)
		router := handler.BuildRouter()
		if *tlscert == "" && *tlskey == "" {
			log.Fatal(http.ListenAndServe(*address, router))
		} else {
			log.Fatal(http.ListenAndServeTLS(*address, *tlscert, *tlskey, router))
		}
	}()

	waitForExit(handler.Stop)
}

func waitForExit(cb func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c
	cb()
	time.Sleep(500 * time.Millisecond)
}
