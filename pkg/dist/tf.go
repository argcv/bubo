package dist

import "sync"

type TicketFactory struct {
	c uint64
	m *sync.Mutex
}

func (t *TicketFactory) Get() uint64 {
	t.m.Lock()
	defer t.m.Unlock()
	t.c ++
	return t.c
}

func NewTicketFactory() *TicketFactory {
	return &TicketFactory{
		c: 0,
		m: &sync.Mutex{},
	}
}
