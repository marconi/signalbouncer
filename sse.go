package signalbouncer

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

var (
	newline    = "\n"
	sseHeaders = map[string]string{
		"Connection":    "keep-alive",
		"Content-Type":  "text/event-stream",
		"Cache-Control": "no-cache",
		corsAllowOrigin: "*",
	}
)

type SSESignalHandler struct {
	peer            *Peer
	dataCounter     int64
	dataChan        chan string
	stopChan        chan bool
	handlerStopChan chan<- string
}

func NewSSESignalHandler(peer *Peer, handlerStopChan chan<- string) *SSESignalHandler {
	handler := &SSESignalHandler{
		peer:            peer,
		dataChan:        make(chan string),
		stopChan:        make(chan bool),
		handlerStopChan: handlerStopChan,
	}

	// Subscribe to data coming in from peer
	peer.Subscribe(handler.dataChan)

	return handler
}

func (sse *SSESignalHandler) Serve(w http.ResponseWriter, r *http.Request) {
	closeNotifier := w.(http.CloseNotifier).CloseNotify()

	// send headers
	writeHeaders(w, sseHeaders)
	w.(http.Flusher).Flush()

	// send peer id
	sse.dataCounter += int64(1)
	fmt.Fprint(w, sseFormatData(sse.dataCounter, "peerId", sse.peer.Id))
	w.(http.Flusher).Flush()

	// poll for data
	for {
		select {
		case data := <-sse.dataChan:
			log.WithFields(log.Fields{"peerId": sse.peer.Id}).Infof("received data:\n%s\n", data)
			sse.dataCounter += int64(1)
			fmt.Fprint(w, sseFormatData(sse.dataCounter, "signal", data))
			w.(http.Flusher).Flush()
		case <-closeNotifier:
			go sse.Stop()
		case <-sse.stopChan:
			return
		}
	}
}

func (sse *SSESignalHandler) Stop() {
	log.WithFields(log.Fields{"peerId": sse.peer.Id}).Info("stopping sse signal handler")
	sse.peer.Unsubscribe(sse.dataChan)
	sse.stopChan <- true
	close(sse.stopChan)
	close(sse.dataChan)
	sse.handlerStopChan <- sse.peer.Id
}

func sseFormatData(id int64, event, data string) string {
	var output []string

	if id > 0 {
		output = append(output, fmt.Sprintf("id: %d", id))
	}
	if event != "" {
		output = append(output, fmt.Sprintf("event: %s", event))
	}

	datas := strings.Split(data, newline)
	for i, data := range datas {
		datas[i] = fmt.Sprintf("data: %s", data)
	}
	output = append(output, strings.Join(datas, newline))
	return strings.Join(output, newline) + strings.Repeat(newline, 2)
}
