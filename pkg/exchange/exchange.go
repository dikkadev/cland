package exchange

type Notification struct {
	id       int
	Topic    string
	Metadata map[string]string
	Message  string
}
