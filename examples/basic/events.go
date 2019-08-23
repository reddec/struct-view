package basic

import "sync"

type UserCreated struct {
	lock     sync.RWMutex
	handlers []func(User)
}

func (ev *UserCreated) Subscribe(handler func(User)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *UserCreated) Emit(payload User) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type UserRemoved struct {
	lock     sync.RWMutex
	handlers []func(User)
}

func (ev *UserRemoved) Subscribe(handler func(User)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *UserRemoved) Emit(payload User) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type UserSubscribed struct {
	lock     sync.RWMutex
	handlers []func(Subscription)
}

func (ev *UserSubscribed) Subscribe(handler func(Subscription)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *UserSubscribed) Emit(payload Subscription) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}

type UserLeaved struct {
	lock     sync.RWMutex
	handlers []func(Subscription)
}

func (ev *UserLeaved) Subscribe(handler func(Subscription)) {
	ev.lock.Lock()
	ev.handlers = append(ev.handlers, handler)
	ev.lock.Unlock()
}
func (ev *UserLeaved) Emit(payload Subscription) {
	ev.lock.RLock()
	for _, handler := range ev.handlers {
		handler(payload)
	}
	ev.lock.RUnlock()
}
