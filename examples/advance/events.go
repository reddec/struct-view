package advance

import "sync"

type eventUserCreated struct {
	lock     sync.RWMutex
	handlers []func(User)
}

func (ev *eventUserCreated) Subscribe(handler func(User)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventUserCreated) Emit(payload User) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type eventUserRemoved struct {
	lock     sync.RWMutex
	handlers []func(User)
}

func (ev *eventUserRemoved) Subscribe(handler func(User)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventUserRemoved) Emit(payload User) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type eventUserSubscribed struct {
	lock     sync.RWMutex
	handlers []func(Subscription)
}

func (ev *eventUserSubscribed) Subscribe(handler func(Subscription)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventUserSubscribed) Emit(payload Subscription) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type eventUserLeaved struct {
	lock     sync.RWMutex
	handlers []func(Subscription)
}

func (ev *eventUserLeaved) Subscribe(handler func(Subscription)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *eventUserLeaved) Emit(payload Subscription) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type Events struct {
	UserCreated    eventUserCreated
	UserRemoved    eventUserRemoved
	UserSubscribed eventUserSubscribed
	UserLeaved     eventUserLeaved
}

func (bus *Events) Emitter() *emitterEvents {
	return &emitterEvents{events: bus}
}

type emitterEvents struct {
	events *Events
}

func (emitter *emitterEvents) UserCreated(payload User) {
	emitter.events.UserCreated.Emit(payload)
}
func (emitter *emitterEvents) UserRemoved(payload User) {
	emitter.events.UserRemoved.Emit(payload)
}
func (emitter *emitterEvents) UserSubscribed(payload Subscription) {
	emitter.events.UserSubscribed.Emit(payload)
}
func (emitter *emitterEvents) UserLeaved(payload Subscription) {
	emitter.events.UserLeaved.Emit(payload)
}
