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
	address    = flag.String("address", "127.0.0.1:8080", "Set listening address")
	configFile = flag.String("configFile", "config.json", "Path to config file")
)

func main() {
	flag.Parse()

	config := signalbouncer.LoadConfig(*configFile)
	signalRooms := signalbouncer.NewSignalRooms()
	signalService := signalbouncer.NewSignalService(config)
	handler := signalbouncer.NewHandler(signalRooms, signalService)

	go func() {
		log.Infof("listening on %s", *address)
		router := handler.BuildRouter()
		if config.TlsCert == "" && config.TlsKey == "" {
			log.Fatal(http.ListenAndServe(*address, router))
		} else {
			log.Fatal(http.ListenAndServeTLS(*address, config.TlsCert, config.TlsKey, router))
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
