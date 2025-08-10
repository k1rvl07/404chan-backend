package utils

import (
	"sync"
)

type Event struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

type Handler func(event Event)

type EventBus struct {
	subscribers map[string][]Handler
	events      chan Event
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]Handler),
		events:      make(chan Event, 100),
	}
}

func (eb *EventBus) Publish(event string, data interface{}) {
	e := Event{Event: event, Data: data}
	select {
	case eb.events <- e:
	default:
	}
}

func (eb *EventBus) Subscribe(event string, handler Handler) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[event] = append(eb.subscribers[event], handler)
}

func (eb *EventBus) SubscribeCh() <-chan Event {
	return eb.events
}
