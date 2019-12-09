package events

//go:generate mockgen -destination=dispatcher_mocks.go -package events -self_package github.com/teserakt-io/c2/internal/events github.com/teserakt-io/c2/internal/events Dispatcher

import (
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Dispatcher defines a component able to dispatch an event to
// all subscribed listeners
type Dispatcher interface {
	AddListener(Listener)
	RemoveListener(Listener) error
	Listeners() []Listener
	Dispatch(Event)
}

type dispatcher struct {
	logger    log.FieldLogger
	listeners []Listener
	lock      sync.RWMutex
}

var _ Dispatcher = (*dispatcher)(nil)

// NewDispatcher returns a new instance of an event dispatcher
func NewDispatcher(logger log.FieldLogger) Dispatcher {
	return &dispatcher{
		logger: logger,
	}
}

// AddListener register the given listener on the dispatcher, making it ready to receive events
func (d *dispatcher) AddListener(lis Listener) {
	d.lock.Lock()
	d.listeners = append(d.listeners, lis)
	d.lock.Unlock()

	d.logger.WithField("listener", fmt.Sprintf("%p", lis)).Info("registered new listener on event dispatcher")
}

// Listeners returns the list of registered listeners on the dispatcher
func (d *dispatcher) Listeners() []Listener {
	return d.listeners
}

// RemoveListener will remove given listener from the dispatcher listeners.
// or return ErrListenerNotFound when the listener is not registered on this
// dispatcher.
func (d *dispatcher) RemoveListener(l Listener) error {
	for i, lis := range d.listeners {
		if lis == l {
			d.lock.Lock()
			d.listeners = append(d.listeners[:i], d.listeners[i+1:]...)
			d.lock.Unlock()

			d.logger.WithField("listener", fmt.Sprintf("%p", lis)).Info("removed listener from event dispatcher")
			return nil
		}
	}

	return ErrListenerNotFound
}

// Dispatch will fan out the provided event to every registered listerners
func (d *dispatcher) Dispatch(evt Event) {
	d.lock.RLock()
	for _, lis := range d.listeners {
		lis.Send(evt)
	}
	d.logger.WithField("count", len(d.listeners)).Debug("dispatched event to listeners")
	d.lock.RUnlock()
}
