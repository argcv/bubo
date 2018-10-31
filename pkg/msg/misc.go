package msg

import "sync"

type M map[string]interface{}

type State int

const (
	StCreate State = iota + 1
	StPending
	StRunning
	StShuttingDown
	StStopped = StCreate
)

func UpdateState(mtx *sync.Mutex, st func() *State, dest State, src ... State) bool {
	mtx.Lock()
	defer mtx.Unlock()

	cst := st()

	for _, f := range src {
		if f == *cst {
			*cst = dest
			return true
		}
	}
	return false
}
