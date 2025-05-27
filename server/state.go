package main

import (
	"net"
	"sync"
)

type Client struct {
	Addr     *net.UDPAddr
	Nickname string
	Channel  string
}

type ServerState struct {
	sync.Mutex
	Clients map[string]*Client // nickname -> Client
}

var state = &ServerState{
	Clients: make(map[string]*Client),
}

// Attempts to reserve a nickname. Returns true if successful.
func reserveNickname(nick string, addr *net.UDPAddr) bool {
	state.Lock()
	defer state.Unlock()

	if _, exists := state.Clients[nick]; exists {
		return false
	}

	state.Clients[nick] = &Client{
		Addr:     addr,
		Nickname: nick,
		Channel:  "General", // default channel
	}
	return true
}

func channelExists(name string) bool {
	for _, ch := range serverConfig.Channels {
		if ch.Name == name {
			return true
		}
	}
	return false
}

func updateClientChannel(addr *net.UDPAddr, channel string) bool {
	state.Lock()
	defer state.Unlock()
	for _, client := range state.Clients {
		if client.Addr.String() == addr.String() {
			client.Channel = channel
			return true
		}
	}
	return false
}

// Returns a list of all current nicknames
func listNicknames() []string {
	state.Lock()
	defer state.Unlock()

	nicks := make([]string, 0, len(state.Clients))
	for nick := range state.Clients {
		nicks = append(nicks, nick)
	}
	return nicks
}
