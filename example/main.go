package dbuf

import (
	"net/http"
	"log"
	"os"
	"fmt"
	"io/ioutil"
	"bytes"
	"strings"
	"io"

	"github.com/zieckey/go-doublebuffering"
)

type BlackIDDict struct {
	blackIDs map[string]int
}

func NewBlackIDDict() dbuf.DoubleBufferingTarget {
	d := &BlackIDDict{
		blackIDs: make(map[string]int),
	}
	return d
}

var dbm *dbuf.DoubleBufferingManager

func (d *BlackIDDict) Initialize(conf string) bool {
	filepath := conf

	// 加载黑名单列表文件，每行一个
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return false
	}
	r := bytes.NewBuffer(b)
	for {
		id, err := r.ReadString('\n')
		if err == io.EOF || err == nil {
			id = strings.TrimSpace(id)
			if len(id) > 0 {
				d.blackIDs[id] = 1
			}
		}

		if err != nil {
			break
		}
	}

	return true
}

func (d *BlackIDDict) Close() {
	// 在这里做一些资源释放工作
	// 当前这个例子没有资源需要我们手工释放
}

func (d *BlackIDDict) IsBlackID(id string) bool {
	_, exist := d.blackIDs[id]
	return exist
}

func Query(r *http.Request) (string, error) {
	id := r.FormValue("id")
	query := r.FormValue("query")

	//TODO 参数合法性检查

	d := dbm.Get("black_id")
	tg := d.Get()
	defer tg.Release()
	dict := tg.Target.(*BlackIDDict)  // 转换为具体的Dict对象
	if dict == nil {
		return "", fmt.Errorf("ERROR, Convert DoubleBufferingTarget to Dict failed")
	}

	if dict.IsBlackID(id) {
		return "ERROR", fmt.Errorf("ERROR id")
	}

	//具体的业务逻辑，查询数据库/NoSQL等数据引擎，然后做逻辑计算，然后合并结果
	//这里简单抽象，直接返回欢迎语
	result := fmt.Sprintf("hello, %v", id)

	// 记录一条查询日志，用于离线统计和分析
	log.Printf("<id=%v><query=%v><result=%v><ip=%v>", id, query, result, r.RemoteAddr)

	return result, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	result, err := Query(r)
	if err == nil {
		w.Write([]byte(result))
	} else {
		w.WriteHeader(403)
		w.Write([]byte(result))
	}
}

func Reload(w http.ResponseWriter, r *http.Request) {
	// 这里简化处理，直接重新加载black_id。如果有多个，可以从url参数中获取资源名称、资源所在的路径
	if dbm.Reload("black_id", os.Args[1]) {
		w.Write([]byte("OK"))
	} else {
		w.Write([]byte("FAILED"))
	}
}

func main() {
	if len(os.Args) != 2 {
		panic("Not specify black_id.txt")
	}

	dbm = dbuf.NewDoubleBufferingManager()
	rc := dbm.Add("black_id", os.Args[1], NewBlackIDDict)
	if rc == false {
		panic("black_id initialize failed")
	}

	http.HandleFunc("/q", Handler)
	http.HandleFunc("/admin/reload", Reload) // 管理接口，用于重新加载black_id.txt。如果有多个这种资源，可以增加一些参数来说区分不同的资源
	hostname, _ := os.Hostname()
	log.Printf("start http://%s:8091/q", hostname)
	log.Fatal(http.ListenAndServe(":8091", nil))
}

