package dbuf

import (
	"sync"
	"io/ioutil"
	"crypto/md5"
	"fmt"
	"time"
	"sync/atomic"
)

// DoubleBufferingTarget is the interface that wraps the basic operations
// on a resource.
//
// Initialize does some initialization operations on the resource.
// When Initialize encounters an error implementations must return false.
//
// Close does some resource recycling works which cannot be done
// by GC of Golang.
type Target interface {
	Initialize(conf string) bool
	Close()
}

type TargetCreator func() Target

type TargetRef struct {
	Target Target
	ref    *int32
}

type DoubleBuffering struct {
	creator         TargetCreator

	mutex           sync.Mutex
	refTarget       TargetRef

	reloadTimestamp int64
	md5h            string
}


func newDoubleBuffering(f TargetCreator) *DoubleBuffering {
	d := new(DoubleBuffering)
	d.creator = f
	d.reloadTimestamp = 0
	return d
}

func (d *DoubleBuffering) reload(conf string) bool {
	t := d.creator()
	if t.Initialize(conf) == false {
		return false
	}

	content, err := ioutil.ReadFile(conf)
	if err != nil {
		content = []byte(conf)
	}
	d.md5h = fmt.Sprint("%x", md5.Sum(content))
	d.reloadTimestamp = time.Now().Unix()

	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.refTarget.Release() // 将老对象释放掉

	d.refTarget.Target = t
	d.refTarget.ref = new(int32)
	*d.refTarget.ref = 1 // 初始设置为1，由DoubleBuffering代为管理

	return true
}

// ReloadTimestamp returns the latest timestamp when the DoubleBuffering reloaded at the last time
func (d *DoubleBuffering) ReloadTimestamp() int64 {
	return d.reloadTimestamp
}

// LatestConfMD5 returns the latest config's md5
func (d *DoubleBuffering) LatestConfMD5() string {
	return d.md5h
}

// Get returns the target this DoubleBuffering manipulated.
// You should call TargetRef.Release() function after you have used it.
func (d *DoubleBuffering) Get() TargetRef {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	atomic.AddInt32(d.refTarget.ref, 1)
	return d.refTarget
}

// Release decrease one reference count.
func (d TargetRef) Release() {
	if d.ref != nil && atomic.AddInt32(d.ref, -1) == 0 {
		d.Target.Close()
	}
}

// Ref returns the reference count of the resource.
func (d TargetRef) Ref() int32 {
	if d.ref != nil {
		return *d.ref
	}

	return 0
}

type dbmap map[string/*name*/]*DoubleBuffering
type Manager struct {
	targets dbmap
	mutex   sync.Mutex
}

func NewManager() *Manager {
	m := new(Manager)
	m.targets = make(dbmap)
	return m
}

func (m *Manager) Add(name string, conf string, f TargetCreator) bool {
	d := newDoubleBuffering(f)
	if d.reload(conf) {
		m.targets[name] = d
		return true
	}

	return false
}

func (m *Manager) Get(name string) *DoubleBuffering {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if t, ok := m.targets[name]; ok {
		return t
	}

	return nil
}

func (m *Manager) Reload(name, conf string) bool {
	d := m.Get(name)
	if d == nil {
		return false
	}

	return d.reload(conf)
}
