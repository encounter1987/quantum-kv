package quantumdb

import "github.com/hashicorp/raft"

type fsmSnapshot struct {
	kv map[string]string
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {

	return nil
}

func (f *fsmSnapshot) Release() {
	
}
