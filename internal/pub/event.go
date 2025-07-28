package pub

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}
