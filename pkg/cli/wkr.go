package cli

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/argcv/picidae/pkg/msg"
	"github.com/argcv/picidae/pkg/utils"
	"github.com/argcv/stork/log"
	"github.com/pkg/errors"
	"io"
	"net"
	"sync"
	"time"
)

type Worker struct {
	ID uint64

	Conn net.Conn

	Links map[uint64]*Link

	LocalHost  string
	LocalPort  int
	RemotePort int

	Secret string

	MsgQOut chan msg.Message
	MsgQIn  chan msg.Message

	Mgr *Manager

	LastUpdate time.Time

	m               *sync.Mutex
	wg              *sync.WaitGroup
	stop            chan bool
	stopRecv        chan bool
	stopHealthCheck chan bool
	state           msg.State
}

func (w *Worker) Ctx() context.Context {
	return w.Mgr.ctx
}

func NewWorker(mgr *Manager, lhost string, lport, rport int, secret string) *Worker {
	return &Worker{
		ID: 0,

		Conn: nil,

		Links: map[uint64]*Link{},

		LocalHost:  lhost,
		LocalPort:  lport,
		RemotePort: rport,
		Secret:     secret,

		MsgQOut: nil,
		MsgQIn:  nil,


		Mgr: mgr,

		m:     &sync.Mutex{},
		wg:    &sync.WaitGroup{},
		state: msg.StCreate,
		stop:  nil,
	}
}

func (w *Worker) GetLink(tid uint64) (link *Link, ok bool) {
	w.m.Lock()
	defer w.m.Unlock()
	l, o := w.Links[tid]
	return l, o
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

func (w *Worker) RemoveLink(tid uint64) (err error) {
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

func (w *Worker) Send(m msg.Message) error {
	if w.state != msg.StRunning {
		return errors.New("not started")
	}
	w.MsgQOut <- m
	return nil
}

// This function will establish the connection
// If the 'worker id' is exists, we can continue
// the work, otherwise we may get a new id
func (w *Worker) initConn(c net.Conn, enc *gob.Encoder, dec *gob.Decoder) error {
	c.SetDeadline(time.Now().Add(30 * time.Second))

	// Send connect
	msgEst := msg.EstablishRequest{
		Wid: w.ID, // 0 for default
	}

	// send..
	if e := enc.Encode(msgEst); e != nil {
		errMsg := fmt.Sprintf("Send failed: %v", e.Error())
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	msgEstAck := msg.EstablishAck{}

	if e := dec.Decode(&msgEstAck); e != nil {
		errMsg := fmt.Sprintf("Decode failed: %v", e.Error())
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	w.ID = msgEstAck.Wid

	log.Debugf("ack: %v, worker: %v", msgEstAck, w)

	return nil

}

// update status
func (w *Worker) UpdateState(dest msg.State, src ... msg.State) bool {
	return msg.UpdateState(w.m, func() *msg.State { return &w.state }, dest, src...)
}

func (w *Worker) Start() error {
	if !w.UpdateState(msg.StPending, msg.StStopped, msg.StCreate) {
		return errors.New("already started")
	}
	w.Mgr.wg.Add(1)

	host := w.Mgr.Config.Host
	port := w.Mgr.Config.Port

	linkStr := fmt.Sprintf("%s:%d", host, port)

	conn, err := net.Dial("tcp", linkStr)
	if err != nil {
		log.Fatalf("Start failed: %v", err)
	}

	//seConn := utils.NewSecConn(conn, w.Secret)
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	if e := enc.Encode(msg.NewAuthRequest(w.RemotePort, w.Secret)); e != nil {
		log.Infof("send failed: %v", e)
		conn.Close()
		w.Mgr.wg.Done()
		return e
	}

	rpl := msg.AuthAck{}

	if e := dec.Decode(&rpl); e != nil {
		log.Infof("recv failed: %v", e)
		conn.Close()
		w.Mgr.wg.Done()
		return e
	}

	log.Infof("Connected: %v, repl: %v", w, rpl)

	seConn := utils.NewSecConn(conn, w.Secret)

	w.Conn = seConn

	// New Encoder, Don't close the original
	enc = gob.NewEncoder(seConn)
	dec = gob.NewDecoder(seConn)
	w.MsgQOut = make(chan msg.Message)
	w.MsgQIn = make(chan msg.Message)
	w.stop = make(chan bool)
	w.stopRecv = make(chan bool)
	w.stopHealthCheck = make(chan bool)

	if e := w.initConn(seConn, enc, dec); e != nil {
		log.Infof("Init failed..:%v", e)
	} else {
		log.Infof("Inited..")
		w.state = msg.StRunning
	}

	// receive
	w.wg.Add(1)
	go func() {
		for w.state == msg.StRunning {
			select {
			case <-w.stopRecv:
				log.Infof("Worker:recv: stop signal")
				w.UpdateState(msg.StShuttingDown, msg.StRunning)
			default:
				seConn.SetReadDeadline(time.Now().Add(30 * time.Second))
				msgIn := msg.Message{}
				if e := dec.Decode(&msgIn); e == nil {
					// data received (in)
					w.UpdateTime()
					w.MsgQIn <- msgIn
				} else if e == io.EOF {
					log.Debugf("EOF...")
					continue
				} else {
					log.Warnf("Decode failed: %v", e.Error())
				}
			}
		}

		// gc part
		log.Infof("GC: message receive, started")
		close(w.MsgQIn)
		w.wg.Done()
		log.Infof("GC: message receive, ok")
	}()

	// loop and process messages
	w.wg.Add(1)
	go func() {
		for w.state == msg.StRunning {
			select {
			case cMsg, ok := <-w.MsgQIn:
				// receive message
				if !ok {
					log.Infof("Worker:msg:in: closed")
					w.UpdateState(msg.StShuttingDown, msg.StRunning)
				} else {
					log.Debugf("Worker:msg:in: %v", cMsg)
					// get new message
					switch cMsg.Meta.Op {
					case msg.OpEstablish:
						// New Link
						msgEstAck := msg.Message{
							Meta: msg.Meta{
								Op:      msg.OpEstablishAck,
								Status:  true,
								Wid:     w.ID,
								Seq:     cMsg.Meta.Seq + 1,
								Message: "",
							},
						}

						if e := w.AddLink(cMsg.Meta.Tid); e != nil {
							log.Errorf("Add link failed: %v", e)
							msgEstAck.Meta.Status = false
							msgEstAck.Meta.Message = e.Error()
						}

						w.Send(msgEstAck)
					case msg.OpAliveAck:
						cMsg.Meta.Seq += 1
						w.Send(cMsg)
					case msg.OpRead:
						tid := cMsg.Meta.Tid
						if link, ok := w.GetLink(tid); ok {
							link.MsgIn(cMsg)
						} else {
							msgReadAck := msg.Message{
								Meta: msg.Meta{
									Op:      msg.OpReadAck,
									Status:  false,
									Wid:     w.ID,
									Tid:     tid,
									Seq:     cMsg.Meta.Seq + 1,
									Message: "not_found",
								},
							}
							w.Send(msgReadAck)
						}
					default:
						log.Warnf("Unknown op: %v", cMsg.Meta.Op)
					}
				}
			case cMsg, ok := <-w.MsgQOut:
				if !ok {
					log.Infof("Worker:msg:out: closed")
					w.UpdateState(msg.StShuttingDown, msg.StRunning)
				} else {
					log.Debugf("Worker:msg:out: %v", cMsg)
					seConn.SetWriteDeadline(time.Now().Add(30 * time.Second))
					if e := enc.Encode(cMsg); e != nil {
						log.Errorf("Write message failed: %v", e)
					} else {
						// data sent (out)
						w.UpdateTime()
					}
				}
			case <-time.After(15 * time.Second): // heartbeat
				msgEstAck := msg.Message{
					Meta: msg.Meta{
						Op:     msg.OpAliveAck,
						Status: true,
						Wid:    w.ID,
					},
				}
				log.Debugf("Sending hb")
				if e := w.Send(msgEstAck); e != nil {
					log.Errorf("Send hb failed... %v", e)
				} else {
					log.Debugf("Sent hb")
				}
			case <-w.stop:
				log.Infof("Worker:msg:in: stop signal")
				w.UpdateState(msg.StShuttingDown, msg.StRunning)
			}
		}

		// gc part
		log.Infof("GC: message processing, started")
		close(w.MsgQOut)
		w.wg.Done()
		log.Infof("GC: message processing, ok")
	}()

	w.CheckHealth()

	if w.state == msg.StRunning {
		log.Infof("Worker: started")
		return nil
	} else {
		log.Infof("Worker: not started, waiting to close")
		w.close()
		log.Infof("Worker: not started, closed")
		return errors.New("not started")
	}

}

// send signal, close all channel, update status as stopped
func (w *Worker) close() error {
	w.stopRecv <- true
	w.stopHealthCheck <- true
	w.stop <- true
	close(w.stopRecv)
	close(w.stopHealthCheck)
	close(w.stop)
	w.Conn.Close()
	w.wg.Wait()
	w.Mgr.wg.Done()
	w.state = msg.StStopped
	return nil
}

// try to stop
func (w *Worker) Stop() error {
	if w.UpdateState(msg.StShuttingDown, msg.StRunning) {
		w.close()
		return nil
	} else {
		errMsg := fmt.Sprintf("Unexpected state: %v", w.state)
		log.Error(errMsg)
		return errors.New(errMsg)
	}
}

func (w *Worker) UpdateTime() {
	w.m.Lock()
	defer w.m.Unlock()
	w.LastUpdate = time.Now()
}

func (w *Worker) CheckHealth() {
	w.wg.Add(1)
	go func() {
		running := true
		for running && w.state == msg.StRunning {
			select {
			case <-time.After(1 * time.Minute):
				last := w.LastUpdate
				if last.Add(2 * time.Minute).After(time.Now()) {
					// unhealthy
					log.Warnf("CheckHealth: failed, restarting...")
					w.Restart()
				} else {
					// health
				}
			case <-w.stopHealthCheck:
				running = false
			}

		}
		w.wg.Done()
	}()
}

func (w *Worker) Restart() error {
	log.Warnf("Restarting... %v", w)
	if e := w.Stop(); e != nil {
		log.Errorf("Stop failed: %v", e)
		return e
	}
	if e := w.Start(); e != nil {
		log.Errorf("Start failed: %v", e)
		return e
	}
	log.Warnf("Restarted... %v", w)
	return nil
}

func (w *Worker) String() string {
	return fmt.Sprintf("%v:%v:%v[%v]", w.LocalHost, w.LocalPort, w.RemotePort, w.state)
}
