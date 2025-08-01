package pub

// Event represents a publishable event in the pub/sub system.
// Events are the core message unit that producers publish to topics.
type Event struct {
	// Type identifies the kind of event (e.g., "order.created", "user.updated")
	Type string `json:"type"`
	// Payload contains the event data, can be any JSON-serializable structure
	Payload any `json:"payload"`
}
