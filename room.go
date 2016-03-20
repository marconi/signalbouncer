package signalbouncer

import (
	"errors"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/nu7hatch/gouuid"
)

var (
	ErrGeneratePeerId = errors.New("Error generating peer id")
	ErrInvalidRoom    = errors.New("Invalid room")
	ErrInvalidPeer    = errors.New("Invalid peer")
	ErrReadingSignal  = errors.New("Error reading signal")
)

type Peer struct {
	Id          string `json:"id"`
	dataChan    chan string
	stopChan    chan bool
	subscribers []chan<- string
}

func NewPeer() (*Peer, error) {
	peerId, err := generatePeerId()
	if err != nil {
		return nil, err
	}

	peer := &Peer{
		Id:       peerId,
		dataChan: make(chan string),
		stopChan: make(chan bool),
	}

	go peer.consume()

	return peer, nil
}

func (peer *Peer) Subscribe(subscriber chan<- string) {
	peer.subscribers = append(peer.subscribers, subscriber)
}

func (peer *Peer) Unsubscribe(subscriber chan<- string) {
	for i, subscriber := range peer.subscribers {
		if subscriber == subscriber {
			peer.subscribers = append(peer.subscribers[:i], peer.subscribers[i+1:]...)
		}
	}
}

func (peer *Peer) Send(data string) {
	log.WithFields(log.Fields{"peerId": peer.Id}).Infof("sending to peers:\n%s\n", data)
	peer.dataChan <- data
}

func (peer *Peer) Stop() {
	log.WithFields(log.Fields{"peerId": peer.Id}).Info("stopping peer")
	peer.stopChan <- true
	close(peer.stopChan)
	close(peer.dataChan)
}

// Self consumes from peer's data channel if no subscriber,
// else propagates to subscribers. This is so that peer sending
// doesn't block if there are no subscribers.
func (peer *Peer) consume() {
	for {
		select {
		case data := <-peer.dataChan:
			if len(peer.subscribers) > 0 {
				for _, subscriber := range peer.subscribers {
					subscriber <- data
				}
			}
		case <-peer.stopChan:
			return
		}
	}
}

type Room struct {
	sync.Mutex
	Name  string
	Peers map[string]*Peer
}

func NewRoom(name string) *Room {
	return &Room{
		Name:  name,
		Peers: make(map[string]*Peer),
	}
}

func (room *Room) Join(peer *Peer) {
	room.Lock()
	defer room.Unlock()
	room.Peers[peer.Id] = peer
}

func (room *Room) Emit(peerId, data string) {
	log.WithFields(log.Fields{"peerId": peerId, "roomName": room.Name}).Info("emitting signal")
	for _, peer := range room.Peers {
		if peer.Id != peerId {
			peer.Send(data)
		}
	}
}

func (room *Room) GetPeer(peerId string) *Peer {
	peer, ok := room.Peers[peerId]
	if ok {
		return peer
	}
	return nil
}

type SignalRooms struct {
	sync.Mutex
	rooms map[string]*Room
}

func NewSignalRooms() *SignalRooms {
	return &SignalRooms{
		rooms: make(map[string]*Room),
	}
}

func (signalRooms *SignalRooms) Join(roomName string) (*Peer, error) {
	peer, err := NewPeer()
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"peerId": peer.Id}).Info("peer created")

	signalRooms.Lock()
	defer signalRooms.Unlock()
	room, ok := signalRooms.rooms[roomName]
	if !ok {
		log.WithFields(log.Fields{"roomName": roomName}).Info("room created")
		room = NewRoom(roomName)
	}
	room.Join(peer)
	signalRooms.rooms[roomName] = room

	log.WithFields(log.Fields{"roomName": roomName, "peerId": peer.Id}).Info("joined room")
	return peer, nil
}

func (signalRooms *SignalRooms) Emit(roomName, peerId, data string) {
	room, ok := signalRooms.rooms[roomName]
	if ok {
		room.Emit(peerId, data)
	}
}

func (signalRooms *SignalRooms) Validate(roomName, peerId string) error {
	room, ok := signalRooms.rooms[roomName]
	if !ok {
		return ErrInvalidRoom
	}
	if _, ok = room.Peers[peerId]; !ok {
		return ErrInvalidPeer
	}
	return nil
}

func (signalRooms *SignalRooms) GetRoom(roomName string) *Room {
	room, ok := signalRooms.rooms[roomName]
	if ok {
		return room
	}
	return nil
}

func (signalRooms *SignalRooms) GetPeer(peerId string) *Peer {
	for _, room := range signalRooms.rooms {
		peer, ok := room.Peers[peerId]
		if ok {
			return peer
		}
	}
	return nil
}

func generatePeerId() (string, error) {
	u4, err := uuid.NewV4()
	if err != nil {
		log.Error("unable to generate peer id:", err.Error())
		return "", ErrGeneratePeerId
	}
	return strings.Replace(u4.String(), "-", "", -1), nil
}
