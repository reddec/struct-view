// Code generated by cache-gen. DO NOT EDIT.
//go:generate cache-gen -p users -k int64 -v *UserProfile -I github.com/MyCompany/MyTypes -o user_cached.go
package cache

import (
	"context"
	mytypes "github.com/MyCompany/MyTypes"
	"sync"
)

type UpdaterManager interface {
	Update(ctx context.Context, key int64) (*mytypes.UserProfile, error)
}

type UpdaterManagerFunc func(ctx context.Context, key int64) (*mytypes.UserProfile, error)

func (fn UpdaterManagerFunc) Update(ctx context.Context, key int64) (*mytypes.UserProfile, error) {
	return fn(ctx, key)
}

func NewManagerFunc(updateFunc UpdaterManagerFunc) *Manager {
	return NewManager(updateFunc)
}

func NewManager(updater UpdaterManager) *Manager {
	return &Manager{cache: make(map[int64]*cacheManager), updater: updater}
}

type Manager struct {
	lock    sync.RWMutex
	cache   map[int64]*cacheManager
	updater UpdaterManager
}

func (mgr *Manager) Find(key int64) *cacheManager {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	return mgr.cache[key]
}

func (mgr *Manager) FindOrCreate(key int64) *cacheManager {
	mgr.lock.RLock()
	entry := mgr.cache[key]
	if entry != nil {
		mgr.lock.RUnlock()
		return entry
	}
	mgr.lock.RUnlock()
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	entry = mgr.cache[key]
	if entry != nil {
		return entry
	}
	entry = &cacheManager{key: key, updater: mgr.updater}
	mgr.cache[key] = entry
	return entry
}

func (mgr *Manager) Get(ctx context.Context, key int64) (*mytypes.UserProfile, err) {
	return mgr.FindOrCreate(key).Ensure(ctx)
}

func (mgr *Manager) Set(key int64, value *mytypes.UserProfile) {
	mgr.FindOrCreate(key).Set(value)
}

func (mgr *Manager) Purge(key int64) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	delete(mgr.cache, key)
}

func (mgr *Manager) PurgeAll() {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	mgr.cache = make(map[int64]*cacheManager)
}

func (mgr *Manager) Snapshot() map[int64]*mytypes.UserProfile {
	mgr.lock.RLock()
	defer mgr.lock.RUnlock()
	snapshot := make(map[int64]*mytypes.UserProfile, len(mgr.cache))
	for key, cache := range mgr.cache {
		if !cache.valid {
			continue
		}
		snapshot[key] = cache.data
	}
	return snapshot
}

type cacheManager struct {
	valid   bool
	lock    sync.Mutex
	data    *mytypes.UserProfile
	key     int64
	updater UpdaterManager
}

func (cache *cacheManager) Valid() bool {
	return cache.valid
}

func (cache *cacheManager) Invalidate() {
	cache.valid = false
}

func (cache *cacheManager) Key() int64 {
	return cache.key
}

func (cache *cacheManager) Get() *mytypes.UserProfile {
	return cache.data
}

func (cache *cacheManager) Ensure(ctx context.Context) (*mytypes.UserProfile, error) {
	err := cache.Update(ctx, false)
	return cache.data, err
}

func (cache *cacheManager) Set(value *mytypes.UserProfile) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.data = value
	cache.valid = true
}

func (cache *cacheManager) Update(ctx context.Context, force bool) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cache.valid && !force {
		return nil
	}
	temp, err := cache.updater.Update(ctx, cache.key)
	if err != nil {
		return err
	}
	cache.data = temp
	cache.valid = true
	return nil
}