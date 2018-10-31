package cli

import (
	"fmt"
	"github.com/argcv/picidae/pkg/msg"
	"github.com/argcv/stork/log"
	"github.com/pkg/errors"
	"net"
	"sync"
)

const (
	kMaximumRead = 4096
)

type Link struct {
	ID    uint64
	Local net.Conn // The remote

	Wkr *Worker

	MsgQ chan msg.Message

	m     *sync.Mutex
	wg    *sync.WaitGroup
	stop  chan bool
	state msg.State
}

func NewLink(w *Worker, id uint64, c net.Conn) *Link {
	return &Link{
		ID:    id,
		Local: c,
		Wkr:   w,
		MsgQ:  nil,

		m:     &sync.Mutex{},
		wg:    &sync.WaitGroup{},
		stop:  nil,
		state: msg.StCreate,
	}
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

func (l *Link) MsgOut(m msg.Message) error {
	return l.Wkr.Send(m)
}

func (l *Link) UpdateState(dest msg.State, src ... msg.State) bool {
	return msg.UpdateState(l.m, func() *msg.State { return &l.state }, dest, src...)
}

func (l *Link) Start() error {
	if !l.UpdateState(msg.StPending, msg.StStopped, msg.StCreate) {
		return errors.New("already started")
	}
	log.Debugf("Start a new link: %v => %v", l.Local.LocalAddr(), l.Local.RemoteAddr())
	l.stop = make(chan bool)
	l.MsgQ = make(chan msg.Message)
	l.wg.Add(1)
	l.state = msg.StRunning
	go func() {
		for l.state == msg.StRunning {
			select {
			case cMsg := <-l.MsgQ:
				switch cMsg.Meta.Op {
				case msg.OpRead:
					readLen := cMsg.Len
					if readLen > kMaximumRead {
						readLen = kMaximumRead
					}
					buff := make([]byte, readLen)
					if n, err := l.Local.Read(buff); err != nil {
						log.Errorf("Link:read:failed [%v]", err)
						rMsg := msg.Message{
							Meta: msg.Meta{
								Status:  false,
								Op:      msg.OpReadAck,
								Wid:     l.Wkr.ID,
								Tid:     l.ID,
								Seq:     cMsg.Meta.Seq + 1,
								Message: fmt.Sprintf("Error:%v", err),
							},
						}
						l.MsgOut(rMsg)
					} else {
						rMsg := msg.Message{
							Meta: msg.Meta{
								Status: true,
								Op:     msg.OpReadAck,
								Wid:    l.Wkr.ID,
								Tid:    l.ID,
								Seq:    cMsg.Meta.Seq + 1,
							},
							Data: buff[:n],
							Len:  n,
						}
						l.MsgOut(rMsg)
						log.Debugf("Link:read [%d]", rMsg.Len)
					}
				case msg.OpWrite:
					if n, err := l.Local.Write(cMsg.Data); err != nil {
						rMsg := msg.Message{
							Meta: msg.Meta{
								Status:  false,
								Op:      msg.OpWriteAck,
								Wid:     l.Wkr.ID,
								Tid:     l.ID,
								Seq:     cMsg.Meta.Seq + 1,
								Message: fmt.Sprintf("Error:%v", err),
							},
						}
						l.MsgOut(rMsg)
						log.Errorf("Link:write:failed [%v]", err)
					} else {
						rMsg := msg.Message{
							Meta: msg.Meta{
								Status:  true,
								Op:      msg.OpWriteAck,
								Wid:     l.Wkr.ID,
								Tid:     l.ID,
								Seq:     cMsg.Meta.Seq + 1,
							},
							Len: n,
						}
						l.MsgOut(rMsg)
						log.Debugf("Link:write [%d]", rMsg.Len)
					}
				case msg.OpClose:
					log.Debugf("Link:op:close [%v]", cMsg.Meta.Op)
					rMsg := msg.Message{
						Meta: msg.Meta{
							Status:  true,
							Op:      msg.OpCloseAck,
							Wid:     l.Wkr.ID,
							Tid:     l.ID,
							Seq:     cMsg.Meta.Seq + 1,
						},
					}
					l.MsgOut(rMsg)
					log.Debugf("Link:write [%d]", rMsg.Len)
					l.Stop()
					break
				default:
					log.Warnf("Unknown op: %v", cMsg.Meta.Op)
				}
			case <-l.stop:
				log.Debugf("Link:sig:close")
				l.Stop()
				break
			}
		}
		close(l.MsgQ)
		l.wg.Done()
	}()
	return nil
}

func (l *Link) Stop() error {
	if l.UpdateState(msg.StShuttingDown, msg.StRunning) {
		// Call for removing self
		l.Wkr.RemoveLink(l.ID)

		log.Debugf("Send stop signal...")
		l.stop <- true
		log.Debugf("Close channel")
		close(l.stop)
		log.Debugf("Waiting...")
		l.wg.Wait()
		log.Debugf("Closed")
		l.state = msg.StStopped
		return l.Local.Close()
	} else {
		errMsg := fmt.Sprintf("Unexpected state: %v", l.state)
		log.Error(errMsg)
		return errors.New(errMsg)
	}
}
