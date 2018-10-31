package srv

import (
	"context"
	"encoding/gob"
	"fmt"
	"github.com/argcv/picidae/assets"
	"github.com/argcv/picidae/pkg/msg"
	"github.com/argcv/stork/log"
	"github.com/pkg/errors"
	"net"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	Services map[int]*Service //port

	Config *assets.PicidaeSrvConfig

	m       *sync.Mutex
	wg      *sync.WaitGroup
	ctx     context.Context
	stop    chan bool
	started bool
}

func NewManager(cfg *assets.PicidaeSrvConfig) (m *Manager, err error) {
	m = &Manager{
		Services: map[int]*Service{},
		Config:   cfg,

		m:       &sync.Mutex{},
		wg:      &sync.WaitGroup{},
		ctx:     context.Background(),
		stop:    make(chan bool),
		started: false,
	}

	for _, s := range cfg.Services {
		if e := m.AddService(s.Port, s.Secret); e != nil {
			log.Errorf("Add service failed: %v", e)
		} else {
			log.Infof("Add service: %v", s.Port)
		}
	}

	return
}

func (m *Manager) AddService(port int, secret string) error {

	if _, ok := m.Services[port]; ok {
		errMsg := fmt.Sprintf("Add service failed: %v, already exist", port)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	m.Services[port] = NewService(m, port, secret)

	return nil
}

func (m *Manager) Switch(c net.Conn) {
	if m.started == false {
		log.Errorf("omit request: stopping..")
		return
	}

	valid := false
	defer func() {
		if !valid {
			log.Infof("Close: %v => %v", c.LocalAddr(), c.RemoteAddr())
			c.Close()
		}
	}()

	//defer c.Close()
	c.SetDeadline(time.Now().Add(5 * time.Second))
	// if never timeout
	//c.SetDeadline(time.Unix(0, 0))
	enc := gob.NewEncoder(c)
	dec := gob.NewDecoder(c)

	svg := msg.AuthRequest{}

	if err := dec.Decode(&svg); err != nil {
		log.Errorf("read svg failed: %v", err)
	} else {
		m.m.Lock()
		srv, ok := m.Services[svg.Port]
		m.m.Unlock()

		if !ok {
			errMsg := "svg.not_found"
			log.Debug(errMsg)
			rpl := &msg.AuthAck{
				Success: false,
				Message: errMsg,
			}
			enc.Encode(rpl)
			return
		}

		if ! svg.CheckSecret(srv.Secret) {
			errMsg := "token.invalid"
			log.Debug(errMsg)
			rpl := &msg.AuthAck{
				Success: false,
				Message: errMsg,
			}
			enc.Encode(rpl)
			return
		}

		// Confirm: is trusted worker
		rpl := &msg.AuthAck{
			Success: true,
			Message: "ok",
		}
		enc.Encode(rpl)

		srv.AddWorker(c)

	}
}

func (m *Manager) checkAndStart() bool {
	m.m.Lock()
	defer m.m.Unlock()
	if m.started {
		log.Errorf("is started...")
		return false
	}
	m.started = true
	return true
}

func (m *Manager) startServices() error {
	for _, srv := range m.Services {
		if err := srv.Start(); err == nil {
			log.Infof("Started: %v", srv.Port)
		} else {
			log.Errorf("Start %v failed: %v", srv.Port, err)
			return err
		}
	}
	return nil
}

func (m *Manager) stopServices() (err error) {
	log.Errorf("Stopping all services...")
	errs := []string{}
	for _, srv := range m.Services {
		log.Infof("Shutting down: %v", srv.Port)
		if e := srv.Stop(); e != nil {
			log.Errorf("Shutdown %v failed: %v", srv.Port, e)
			errs = append(errs, e.Error())
		}
	}
	if len(errs) == 0 {
		return nil
	} else {
		return errors.New(strings.Join(errs, ";"))
	}
}

func (m *Manager) Start() {
	if !m.checkAndStart() {
		return
	}

	if err := m.startServices(); err != nil {
		log.Errorf("Start services failed: %v", err)
		m.started = false
		return
	}

	listenStr := fmt.Sprintf("%s:%d", m.Config.Bind, m.Config.Port)
	l, err := net.Listen("tcp4", listenStr)
	if err != nil {
		log.Errorf("Listen aborted: %v", err)
		m.started = false
		return
	}
	defer l.Close()

	log.Infof("Listening: %v", listenStr)

	cc := make(chan net.Conn)

	running := true

	go func() {
		for running {
			c, err := l.Accept()
			if err != nil {
				log.Errorf("Accept failed: %v", err)
				return
			} else {
				log.Debugf("New connection %v => %v", c.RemoteAddr(), c.LocalAddr())
			}
			cc <- c
		}
		close(cc)
	}()

	for running {
		select {
		case c, ok := <-cc:
			if !ok {
				running = false
				log.Infof("Manager Closed")
				break
			} else {
				go m.Switch(c)
			}
		case <-m.stop:
			running = false
			log.Infof("Signal: Stop")
			break
		}
	}
	m.stopServices()
}

func (m *Manager) shutdown() {
	if m.started {
		// send signal
		m.stop <- true
		// stop
		m.wg.Wait()
	}
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.ctx = ctx
	stop := make(chan bool)
	go func() {
		m.shutdown()
		stop <- true
	}()

	select {
	case <-ctx.Done():
		log.Errorf("Force finished: %v", ctx.Err())
		break
	case <-stop:
		log.Infof("finished")
	}
	m.started = false
	return nil
}
