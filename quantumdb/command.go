package quantumdb

type Type string

const (
	PUT    Type = "PUT"
	DELETE Type = "DELETE"
)

type command struct {
	Type Type   `json:"op,omitempty"`
	Data []byte `json:"data,omitempty"`
}
