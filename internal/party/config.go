package party

type (
	lobbyType int
)

const (
	PrivateLobby lobbyType = iota
	PublicLobby
)

type Config struct {
	Visibility    lobbyType
	HasPassphrase bool
	AllowOnlyHost bool
	Passphrase    string
}
