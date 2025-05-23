package quantumdb

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	"io"
	"maps"
)

type fsm KVStore

func (f *fsm) Apply(l *raft.Log) interface{} {
	var cmd command
	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		return fmt.Errorf("error unmarshalling command: %s", err)
	}

	switch cmd.Type {
	case PUT:
		// Unmarshal the data into a key-value pair
		kvData := make(map[string]string)
		if err := json.Unmarshal(cmd.Data, &kvData); err != nil {
			return fmt.Errorf("error unmarshalling command data: %s", err)
		}

		var key, value string
		for key, value = range kvData {
			break
		}

		return f.applyPut(key, value)

	case DELETE:
		// Unmarshal the data into a key
		return f.applyDelete(string(cmd.Data))
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	currentSnapshot := maps.Clone(f.kv)

	return &fsmSnapshot{kv: currentSnapshot}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	oldData := make(map[string]string)

	if err := json.NewDecoder(rc).Decode(&oldData); err != nil {
		return fmt.Errorf("error decoding snapshot data: %s", err)
	}

	f.kv = oldData
	return nil
}

func (f *fsm) applyPut(key, value string) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.kv[key] = value
	return nil
}

func (f *fsm) applyDelete(key string) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.kv, key)
	return nil
}
