package srv

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/argcv/picidae/pkg/dist"
	"github.com/argcv/picidae/pkg/msg"
	"github.com/argcv/picidae/pkg/utils"
	"github.com/argcv/stork/log"
	"net"
	"sync"
	"time"
)

type Service struct {
	Workers map[uint64]*Worker

	Links map[uint64]*Link

	Port   int
	Secret string

	Mgr *Manager

	m  *sync.Mutex
	tf *dist.TicketFactory
}

func (w *Service) Ctx() context.Context {
	return w.Mgr.ctx
}

func NewService(mgr *Manager, port int, secret string) *Service {
	return &Service{
		Workers: map[uint64]*Worker{},
		Links:   map[uint64]*Link{},

		Port:   port,
		Secret: secret,

		Mgr: mgr,
		m:   &sync.Mutex{},
		tf:  dist.NewTicketFactory(),
	}
}

func (srv *Service) AddWorker(c net.Conn) error {
	seConn := utils.NewSecConn(c, srv.Secret)
	// 5 more seconds, add to worker candidate
	seConn.SetDeadline(time.Now().Add(30 * time.Second))

	enc := gob.NewEncoder(seConn)
	dec := gob.NewDecoder(seConn)

	// receive
	msgEst := msg.EstablishRequest{}

	if e := dec.Decode(&msgEst); e != nil {
		errMsg := fmt.Sprintf("Recv failed: %v", e.Error())
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	// check worker

	var wkr *Worker = nil

	if msgEst.Wid == 0 {
		newWid := srv.tf.Get()
		wkr = NewWorker(srv, newWid)
	} else {

	}

	if e := wkr.Attach(seConn, enc, dec); e != nil {

	}

	// msgEstAck := msg.EstablishAck{}

	//sec := a.Secret

	//w := NewWorker(a, c)
	//a.m.Lock()
	//a.Workers = append(a.Workers, w)
	//a.m.Unlock()
	return nil
}

func (w *Worker) AddLink(tid uint64) (err error) {
	log.Debugf("Add link...")
	linkStr := fmt.Sprintf("%s:%d", w.LocalHost, w.LocalPort)
	conn, err := net.Dial("tcp", linkStr)
	if err != nil {
		log.Error(err)
		return err
	}

	link := NewLink(w, tid, conn)

	addLink := func() bool {
		w.m.Lock()
		defer w.m.Unlock()
		if _, ok := w.Links[tid]; ok {
			return false
		}
		w.Links[tid] = link
		return true
	}

	if addLink() {
		return link.Start()
	} else {
		// nothing created if we don't call 'start'
		// just leave it over there, and the
		// gc will help us to clean everything up
		return errors.New("add link failed")
	}
}

// Expected to be called by Link
func (w *Service) RemoveLink(tid uint64) (err error) {
	w.m.Lock()
	defer w.m.Unlock()

	log.Debugf("Remove link...")
	if l, ok := w.Links[tid]; ok {
		delete(w.Links, tid)
		// start removing
		// may return error, since it was removed by itself
		// that's fine
		// just send the requirest
		// and go out
		l.Stop()
		return nil
	} else {
		errMsg := fmt.Sprintf("Link not found: %v", tid)
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}
}


func (srv *Service) Start() error {
	srv.Mgr.wg.Add(1)
	go func() {
		// Process All

		srv.Mgr.wg.Done()
	}()
	return nil
}

func (srv *Service) Stop() error {
	for _, w := range srv.Workers {
		w.Stop()
	}
	srv.Mgr.wg.Wait()
	return nil
}
