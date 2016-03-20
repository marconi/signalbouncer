package signalbouncer

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

var (
	sseProtocol      = "sse"
	wsProtocol       = "websocket"
	allowedProtocols = []string{sseProtocol, wsProtocol}
)

type SignalHandler interface {
	Serve(http.ResponseWriter, *http.Request)
	Stop()
}

type SignalService struct {
	signalHandlers  map[string]SignalHandler
	serviceStopChan chan bool
	handlerStopChan chan string
}

func NewSignalService() *SignalService {
	signalService := &SignalService{
		signalHandlers:  make(map[string]SignalHandler),
		serviceStopChan: make(chan bool),
		handlerStopChan: make(chan string),
	}

	go signalService.handlersWatcher()

	return signalService
}

func (service *SignalService) Validate(protocol string) error {
	for _, allowedProtocol := range allowedProtocols {
		if allowedProtocol == strings.ToLower(protocol) {
			return nil
		}
	}
	return fmt.Errorf("%s: %s", ErrUnsupportedProtocol.Error(), protocol)
}

func (service *SignalService) Serve(w http.ResponseWriter, r *http.Request, peer *Peer, protocol string) {
	switch strings.ToLower(protocol) {
	case sseProtocol:
		log.WithFields(log.Fields{"peerId": peer.Id}).Infof("serving %s", sseProtocol)
		signalHandler := NewSSESignalHandler(peer, service.handlerStopChan)
		service.signalHandlers[peer.Id] = signalHandler
		signalHandler.Serve(w, r)
	}
}

func (service *SignalService) Stop() {
	log.Info("stopping signal handlers")
	for _, handler := range service.signalHandlers {
		handler.Stop()
	}

	log.Info("stopping signal service")
	service.serviceStopChan <- true
	close(service.handlerStopChan)
	close(service.serviceStopChan)
}

func (service *SignalService) handlersWatcher() {
	log.Info("running handler watcher")
	for {
		select {
		case peerId := <-service.handlerStopChan:
			service.removeSignalHandler(peerId)
		case <-service.serviceStopChan:
			return
		}
	}
}

func (service *SignalService) removeSignalHandler(peerId string) {
	log.WithFields(log.Fields{"peerId": peerId}).Info("removing signal handler")
	if _, ok := service.signalHandlers[peerId]; ok {
		delete(service.signalHandlers, peerId)
	}
}
