package common

type ConnectRequest struct {
	Type     string   `json:"type"` // should be "connect"
	Nicklist []string `json:"nicklist"`
}

type ConnectAccepted struct {
	Type       string   `json:"type"` // should be "accept"
	Nickname   string   `json:"nickname"`
	ServerName string   `json:"server_name"`
	MOTD       string   `json:"motd"`
	Channels   []string `json:"channels"`
	Users      []string `json:"users"`
}

type Reject struct {
	Type    string `json:"type"` // "reject"
	Message string `json:"message"`
}