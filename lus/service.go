package lus

/**
  A Service represents the meta data about a Service that is running on the network.
**/

// Represents the JSON data structure that is being passed over the wire to register a service.
type Service struct {
	ID    string // Unique ID for this Service
	Lease int64  // Lease time in ms
	Data  string
	Keys  map[string]string
}

// Initialises and returns a new Client.
func NewService(keys map[string]string, lease int64, data string, id string) Service {
	state := Service{
		Keys:  keys,
		Lease: lease,
		Data:  data,
		ID:    id,
	}
	return state
}
