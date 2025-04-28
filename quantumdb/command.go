package quantumdb

type Type string

const (
	PUT    Type = "PUT"
	DELETE Type = "DELETE"
)

type command struct {
	Type  Type   `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}
