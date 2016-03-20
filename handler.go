package signalbouncer

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
)

var (
	corsAllowOrigin  = "Access-Control-Allow-Origin"
	corsAllowHeaders = "Access-Control-Allow-Headers"
	corsAllowMethods = "Access-Control-Allow-Methods"
	apiHeaders       = map[string]string{
		corsAllowOrigin: "*",
		"Content-Type":  "application/json",
	}
	corsHeaders = map[string]string{
		corsAllowOrigin:  "*",
		corsAllowHeaders: "Origin, X-Requested-With, Content-Type, Accept",
		corsAllowMethods: "GET, POST",
	}

	ErrUnsupportedProtocol = errors.New("Unsupported protocol")
	ErrMarshalingPeer      = errors.New("Error marshaling peer")
)

type Message struct {
	Message string `json:"message"`
}

type Handler struct {
	signalRooms   *SignalRooms
	signalService *SignalService
}

func NewHandler(signalRooms *SignalRooms, signalService *SignalService) *Handler {
	return &Handler{
		signalRooms:   signalRooms,
		signalService: signalService,
	}
}

func (handler *Handler) RoomSignalHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	roomName := ps.ByName("roomName")
	peerId := ps.ByName("peerId")

	log.WithFields(log.Fields{"roomName": roomName, "peerId": peerId}).Info("validating signal")
	if err := handler.signalRooms.Validate(roomName, peerId); err != nil {
		writeError(w, err)
		return
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	defer r.Body.Close()

	room := handler.signalRooms.GetRoom(roomName)
	if room != nil {
		room.Emit(peerId, string(bytes))
	}

	writeMessage(w, "Signal emitted")
}

func (handler *Handler) CORSHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	writeHeaders(w, corsHeaders)
}

func (handler *Handler) StreamHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	protocol := ps.ByName("protocol")
	roomName := ps.ByName("roomName")

	log.WithFields(log.Fields{"protocol": protocol, "roomName": roomName}).Info("handling stream")
	if err := handler.signalService.Validate(protocol); err != nil {
		writeError(w, err)
		return
	}

	peer, err := handler.signalRooms.Join(roomName)
	if err != nil {
		writeError(w, err)
		return
	}

	handler.signalService.Serve(w, r, peer, protocol)
}

func (handler *Handler) BuildRouter() *httprouter.Router {
	router := httprouter.New()
	router.POST("/signal/:roomName/:peerId", handler.RoomSignalHandler)
	router.OPTIONS("/signal/:roomName/:peerId", handler.CORSHandler)
	router.GET("/stream/:protocol/:roomName", handler.StreamHandler)
	return router
}

func (handler *Handler) Stop() {
	handler.signalService.Stop()
}

func writeObject(w http.ResponseWriter, obj interface{}) error {
	writeHeaders(w, apiHeaders)
	bytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	w.Write(bytes)
	return nil
}

func writeMessage(w http.ResponseWriter, msg string) {
	writeHeaders(w, apiHeaders)
	bytes, _ := json.MarshalIndent(&Message{msg}, "", "  ")
	w.Write(bytes)
}

func writeError(w http.ResponseWriter, err error) {
	writeHeaders(w, apiHeaders)
	w.WriteHeader(http.StatusBadRequest)
	bytes, _ := json.MarshalIndent(&Message{err.Error()}, "", "  ")
	w.Write(bytes)
}

func writeHeaders(w http.ResponseWriter, headers map[string]string) {
	for header, value := range headers {
		w.Header().Set(header, value)
	}
}
