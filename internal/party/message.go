package party

const (
	actionRequestState     = "request-state"
	addMessage             = "add-message"
	addActiveConnection    = "add-active-connection"
	removeActiveConnection = "remove-active-connection"
)

type actionMessage struct {
	Action string `json:"action"`
}

type addMessagePayload struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

type activeConnectionUserPayload struct {
	UserName string `json:"user_name"`
	UserID   string `json:"user_id"`
}

type removeConnectionUserPayload struct {
	UserID string `json:"user_id"`
}

type addMessageMessage struct {
	actionMessage
	Payload *addMessagePayload `json:"payload"`
}

type addActiveConnectionMessage struct {
	actionMessage
	Payload *activeConnectionUserPayload `json:"payload"`
}

type removeActiveConnectionMessage struct {
	actionMessage
	Payload *removeConnectionUserPayload `json:"payload"`
}
