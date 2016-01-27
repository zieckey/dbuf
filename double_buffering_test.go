package dbuf_test

import (
	"testing"
	"sync"
	"strconv"
	"time"
	"math/rand"
	"sync/atomic"

	"github.com/bmizerany/assert"

	"golib/dbuf"
)


////////////////////////
// Dict 实现了 unis.DoubleBufferingTarget 接口
type Dict struct {
	d string
	//业务自己的其他更复杂的数据结构
}

func newDict() dbuf.DoubleBufferingTarget {
	d := new(Dict)
	return d
}

var initializedCount int32
var closedCount int32

/*
请求： curl http://localhost:9360/dict
Reload指令：curl "http://localhost:9360/admin/reload?name=mydict&path=xxx2342c"
 */
func (d *Dict) Initialize(conf string) bool {
	// 这个conf一般情况下是一个配置文件的路径
	// 这里我们简单的认为它只是一段数据
	c := atomic.AddInt32(&initializedCount, 1)
	_ = c
	d.d = conf
	//fmt.Printf("Dict.Initialize() called, count=%d\n", c)
	return true
}

func (d *Dict) Close() {
	// 在这里做一些资源释放工作
	c := atomic.AddInt32(&closedCount, 1)
	_ = c
	//fmt.Printf("Dict.Close() called, count=%d\n", c)
}



func TestDoubleBuffering(t *testing.T) {
	m := dbuf.NewDoubleBufferingManager()
	rc := m.Add("mydict", "The config for Dict1", newDict) // 初始化的时候引用计数为1
	assert.Equal(t, rc, true)
	d := m.Get("mydict")
	assert.NotEqual(t, d, nil)
	assert.Equal(t, initializedCount, int32(1))
	assert.Equal(t, closedCount, int32(0))

	var wg sync.WaitGroup
	ReloadedCount := 1000
	for i := 0; i < ReloadedCount; i++ {
		// 模拟一堆协程在同时使用Dict对象
		wg.Add(1)
		go func() {
			defer wg.Done()
			tg := d.Get()
			defer tg.Release()
			dict := tg.Target.(*Dict)
			assert.NotEqual(t, dict, nil)
			assert.Equal(t, tg.Ref() >= 1, true)
			time.Sleep(time.Duration(rand.Intn(100000) + 1) * time.Microsecond)
		}()

		// 模拟不定期的字典文件重新加载
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.Equal(t, m.Reload("mydict", "The new config for Dict" + strconv.Itoa(i)), true)
			time.Sleep(time.Duration(rand.Intn(100000) + 1) * time.Microsecond)
		}()
	}
	wg.Wait()

	assert.Equal(t, initializedCount, int32(ReloadedCount) + 1)
	assert.Equal(t, closedCount, int32(ReloadedCount))

	tg := d.Get()
	defer tg.Release()
	assert.Equal(t, tg.Ref(), int32(2))// 初始化的时候引用计数为1，Get 之后，引用计数又自动加1，因此这里为2。
}