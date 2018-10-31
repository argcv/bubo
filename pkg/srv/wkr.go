package srv

import (
	"encoding/gob"
	"errors"
	"github.com/argcv/picidae/pkg/msg"
	"net"
	"sync"
	"time"
)

type Worker struct {
	ID uint64

	Conn net.Conn
	Dec  *gob.Decoder
	Enc  *gob.Encoder

	NumConn uint64

	MsgQIn  chan msg.Message
	MsgQOut chan msg.Message

	LastUpdated time.Time
	Srv         *Service

	state msg.State
	m     *sync.Mutex
}


func NewWorker(srv *Service, id uint64) *Worker {
	w := &Worker{
		ID: id,

		Conn: nil,
		Enc:  nil,
		Dec:  nil,

		NumConn: 0,

		MsgQIn:  nil,
		MsgQOut: nil,

		LastUpdated: time.Now(),
		Srv:         srv,

		state: msg.StCreate,
		m:     &sync.Mutex{},
	}

	return w
}

func (w *Worker) NumConnIncr(add bool) {
	w.m.Lock()
	defer w.m.Unlock()
	if add {
		w.NumConn ++
	} else {
		w.NumConn --
	}
}

func (w *Worker) UpdateState(dest msg.State, src ... msg.State) bool {
	return msg.UpdateState(w.m, func() *msg.State { return &w.state }, dest, src...)
}

func (w *Worker) Send(m msg.Message) error {
	if w.state != msg.StRunning {
		return errors.New("not started")
	}
	w.MsgQOut <- m
	return nil
}

func (w *Worker) start() {

}

func (w *Worker) Attach(conn net.Conn, enc *gob.Encoder, dec *gob.Decoder) error {

	return nil
}

func (w *Worker) Stop() error {

	return nil
}
