package quantumdb

import (
	"encoding/json"
	"github.com/hashicorp/raft"
)

type fsmSnapshot struct {
	kv map[string]string
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		data, err := json.Marshal(f.kv)
		if err != nil {
			return err
		}

		if _, err := sink.Write(data); err != nil {
			return err
		}

		return sink.Close()
	}()

	if err != nil {
		sink.Cancel()
	}

	return nil
}

func (f *fsmSnapshot) Release() {
	// Clean up resources if needed after the snapshot is released/saved.
}
