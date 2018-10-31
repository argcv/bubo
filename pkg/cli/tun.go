package cli

import (
	"github.com/argcv/picidae/pkg/msg"
	"runtime"
	"sync"
)

type Tunnel struct {
	m     *sync.Mutex
	wg    *sync.WaitGroup
	stop  chan bool
	state msg.State
}

func NewTunnel() *Tunnel {
	t := &Tunnel{

	}

	return t
}

func (t *Tunnel) Send(m msg.Message) error {
	runtime.Gosched()
	return nil
}

func (t *Tunnel) Recv() (*msg.Message) {
	runtime.Gosched()
	return nil
}

func (t *Tunnel) Stop() error {
	return nil
}
