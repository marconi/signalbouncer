package signalbouncer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	log "github.com/Sirupsen/logrus"
)

type IceServer struct {
	Url        string `json:"url"`
	Username   string `json:"username,omitempty"`
	Credential string `json:"credential,omitempty"`
}

type PeerConfig struct {
	IceServers []*IceServer `json:"iceServers"`
}

func (peerConfig *PeerConfig) String() string {
	bytes, err := json.MarshalIndent(peerConfig, "", "  ")
	if err != nil {
		log.Error("error marshaling peer config: ", err.Error())
		return fmt.Sprintf("%+v\n", peerConfig)
	}
	return string(bytes)
}

type Config struct {
	PeerConfig *PeerConfig `json:"peer"`
	TlsCert    string      `json:"tlsCert"`
	TlsKey     string      `json:"tlsKey"`
}

func LoadConfig(filename string) *Config {
	config := new(Config)
	bytes, err := ioutil.ReadFile(filename)
	if err = json.Unmarshal(bytes, config); err != nil {
		log.Warn("error reading config: ", err.Error())
	}
	return config
}
