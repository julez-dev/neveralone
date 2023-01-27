package party

const (
	actionRequestState = "request-state"
	addMessage         = "add-message"
)

type actionMessage struct {
	Action string `json:"action"`
}

type addMessagePayload struct {
	Sender  string `json:"sender"`
	Content string `json:"content"`
}

type addMessageMessage struct {
	actionMessage
	Payload *addMessagePayload `json:"payload"`
}
