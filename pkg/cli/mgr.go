package cli

import (
	"context"
	"errors"
	"github.com/argcv/picidae/assets"
	"github.com/argcv/stork/log"
	"strings"
	"sync"
)

type Manager struct {
	Workers []*Worker //port

	Config *assets.PicidaeCliConfig

	m       *sync.Mutex
	wg      *sync.WaitGroup
	ctx     context.Context
	stop    chan bool
	started bool
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

func (m *Manager) startWorkers() error {
	for _, srv := range m.Workers {
		if err := srv.Start(); err == nil {
			log.Infof("Started: %v", srv)
		} else {
			log.Errorf("Start %v failed: %v", srv, err)
			return err
		}
	}
	return nil
}

func (m *Manager) stopWorkers() (err error) {
	log.Errorf("Stopping all services...")
	errs := []string{}
	for _, srv := range m.Workers {
		log.Infof("Shutting down: %v", srv)
		if e := srv.Stop(); e != nil {
			log.Errorf("Shutdown %v failed: %v", srv, e)
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

	if err := m.startWorkers(); err != nil {
		log.Errorf("Start services failed: %v", err)
		m.started = false
		return
	}

	select {
	case <-m.stop:
		log.Infof("got stop signal")
		m.stopWorkers()
	}
}

func NewManager(cfg *assets.PicidaeCliConfig) (m *Manager, err error) {
	m = &Manager{
		Config: cfg,

		m:       &sync.Mutex{},
		wg:      &sync.WaitGroup{},
		ctx:     context.Background(),
		stop:    make(chan bool),
		started: false,
	}

	for _, cs := range cfg.Services {
		wkr := m.AddWorker(cs.Local.Host, cs.Local.Port, cs.Remote.Port, cs.Secret)
		log.Infof("Add worker: %v", wkr)
	}

	return
}

//

func (m *Manager) AddWorker(lhost string, lport, rport int, secret string) *Worker {
	wkr := NewWorker(m, lhost, lport, rport, secret)
	m.Workers = append(m.Workers, wkr)
	return wkr
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

	return nil
}
