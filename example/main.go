package main

import (
	"net/http"
	"log"
	"os"
	"fmt"
	"io/ioutil"
	"bytes"
	"strings"
	"io"

	"github.com/zieckey/dbuf"
)

type BlackIDDict struct {
	blackIDs map[string]int
}

func NewBlackIDDict() dbuf.Target {
	d := &BlackIDDict{
		blackIDs: make(map[string]int),
	}
	return d
}

var dbm *dbuf.Manager

func (d *BlackIDDict) Initialize(conf string) bool {
	filepath := conf

	// A black name list
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
	// We can do some resource recycling works here if necessarily
}

func (d *BlackIDDict) IsBlackID(id string) bool {
	_, exist := d.blackIDs[id]
	return exist
}

func Query(r *http.Request) (string, error) {
	id := r.FormValue("id")
	query := r.FormValue("query")

	if len(id) == 0 || len(query) == 0 {
		return "", fmt.Errorf("Request parameter error")
	}

	d := dbm.Get("black_id")
	tg := d.Get()
	defer tg.Release()
	dict := tg.Target.(*BlackIDDict)  // convert to concrete object of BlackIDDict
	if dict == nil {
		return "", fmt.Errorf("ERROR, Convert DoubleBufferingTarget to Dict failed")
	}

	if dict.IsBlackID(id) {
		return "ERROR", fmt.Errorf("ERROR id")
	}

	// This is the application business logic, something like query MySQL, calculate, merge result.
	result := fmt.Sprintf("hello, %v", id)

	// Record a query log
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
	r.ParseForm()
	name := r.FormValue("name")
	path := r.FormValue("path")

	if dbm.Reload(name, path) {
		w.Write([]byte("OK"))
	} else {
		w.Write([]byte("FAILED"))
	}
}

// The detail usage is here : http://blog.codeg.cn/2016/01/27/double-buffering/
//
// $ curl "http://127.0.0.1:8091/q?id=475e5a499587a52ea14a23031ecce7c9&query=jane"
// ERROR
//
// $ curl "http://127.0.0.1:8091/q?id=12312&query=jane"
// hello, 12312
func main() {
	if len(os.Args) != 2 {
		panic("Not specify black_id.txt")
	}

	dbm = dbuf.NewManager()
	rc := dbm.Add("black_id", os.Args[1], NewBlackIDDict)
	if rc == false {
		panic("black_id initialize failed")
	}

	http.HandleFunc("/q", Handler)  // query interface
	http.HandleFunc("/admin/reload", Reload) // admin interface
	hostname, _ := os.Hostname()
	log.Printf("start http://%s:8091/q", hostname)
	log.Fatal(http.ListenAndServe(":8091", nil))
}

