package party

type Player struct {
	User   *User
	IsHost bool
}

func NewPlayer(user *User, isHost bool) *Player {
	return &Player{
		User:   user,
		IsHost: isHost,
	}
}
