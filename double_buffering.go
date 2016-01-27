package dbuf

import (
	"sync"
	"io/ioutil"
	"crypto/md5"
	"fmt"
	"time"
	"sync/atomic"
)

type DoubleBufferingTarget interface {
	Initialize(conf string) bool // 初始化，继承类可以在此做一些初始化的工作
	Close() // 继承类如果在Initialize函数中申请了一些资源，可以在这里将这些资源进行回收
}

type DoubleBufferingTargetCreator func() DoubleBufferingTarget

type DoubleBufferingTargetRef struct {
	Target DoubleBufferingTarget
	ref    *int32
}

type DoubleBuffering struct {
	creator         DoubleBufferingTargetCreator

	mutex           sync.Mutex
	refTarget       DoubleBufferingTargetRef

	reloadTimestamp int64
	md5h            string
}


func newDoubleBuffering(f DoubleBufferingTargetCreator) *DoubleBuffering {
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

// ReloadTimestamp return the latest timestamp when the DoubleBuffering reloaded at the last time
func (d *DoubleBuffering) ReloadTimestamp() int64 {
	return d.reloadTimestamp
}

// LatestConfMD5 return the latest config's md5
func (d *DoubleBuffering) LatestConfMD5() string {
	return d.md5h
}

// Get return the target this DoubleBuffering manipulated.
// You should call DoubleBufferingTargetRef.Release() function after you have used it.
func (d *DoubleBuffering) Get() DoubleBufferingTargetRef {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	atomic.AddInt32(d.refTarget.ref, 1)
	return d.refTarget
}

func (d DoubleBufferingTargetRef) Release() {
	if d.ref != nil && atomic.AddInt32(d.ref, -1) == 0 {
		d.Target.Close()
	}
}

func (d DoubleBufferingTargetRef) Ref() int32 {
	if d.ref != nil {
		return *d.ref
	}

	return 0
}

type DoubleBufferingMap map[string/*name*/]*DoubleBuffering
type DoubleBufferingManager struct {
	targets DoubleBufferingMap
	mutex sync.Mutex
}

func NewDoubleBufferingManager() *DoubleBufferingManager {
	m := new(DoubleBufferingManager)
	m.targets = make(DoubleBufferingMap)
	return m
}

func (m *DoubleBufferingManager) Add(name string, conf string, f DoubleBufferingTargetCreator) bool {
	d := newDoubleBuffering(f)
	if d.reload(conf) {
		m.targets[name] = d
		return true
	}

	return false
}

func (m *DoubleBufferingManager) Get(name string) *DoubleBuffering {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if t, ok := m.targets[name]; ok {
		return t
	}

	return nil
}

func (m *DoubleBufferingManager) Reload(name, conf string) bool {
	d := m.Get(name)
	if d == nil {
		return false
	}

	return d.reload(conf)
}
