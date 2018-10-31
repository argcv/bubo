package srv

import (
	"errors"
	"fmt"
	"github.com/argcv/picidae/pkg/msg"
	"github.com/argcv/stork/log"
	"net"
	"sync"
)

const (
	kMaximumRead = 4096
)

type Link struct {
	ID     uint64
	Remote net.Conn // The external client

	Wkr *Worker
	Srv *Service

	MsgQ chan msg.Message

	m     *sync.Mutex
	wg    *sync.WaitGroup
	stop  chan bool
	state msg.State
}

func (l *Link) MsgIn(m msg.Message) error {
	if l.state != msg.StRunning {
		m := fmt.Sprintf("MsgIn: Unexpected state: %v", l.state)
		log.Error(m)
		return errors.New(m)
	}
	log.Debugf("Link:MsgIn start")
	l.MsgQ <- m
	log.Debugf("Link:MsgIn end")
	return nil
}

func (l *Link) Handle(m *msg.Message) {
	switch m.Meta.Op {
	case msg.OpEstablish:
		// ignore
	case msg.OpRead:
		blen := m.Len
		if blen > kMaximumRead {
			blen = kMaximumRead
		}
		buff := make([]byte, blen)
		l.Remote.Read(buff)

	case msg.OpWrite:
	case msg.OpClose:
	}

}

func (l *Link) Stop() error {

	return nil
}
