package pulse

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestHttpRecorder(t *testing.T) {
	hdr := "mongo"
	msg := "Hello World!"
	h := &httpRecorder{}
	f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.w = w
		h.Header().Set(hdr, msg)
		h.WriteHeader(http.StatusTeapot)
		h.Write([]byte(msg))
	})
	s := httptest.NewServer(f)

	r, err := http.Get(s.URL)
	if err != nil {
		t.Fatal("http.Get exits with error:", err)
	}
	defer r.Body.Close()

	if h.c != http.StatusTeapot {
		t.Errorf("httpRecorder failed: Wrong HTTP status code.\nGot : %d\nWant: %d", h.c, http.StatusTeapot)
	}
	if r.Header.Get(hdr) != msg {
		t.Errorf("httpRecorder failed: Header not set.\nGot : %s\nWant: %s", r.Header.Get(hdr), msg)
	}
	if r.StatusCode != http.StatusTeapot {
		t.Errorf("httpRecorder failed: Status code not set.\nGot : %d\nWant: %d", r.StatusCode, http.StatusTeapot)
	}

	o, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Error("Can not read http response body:", err)
	}
	if string(o) != msg {
		t.Errorf("httpRecorder failed: Wrong body wrote.\nGot : %s\nWant: %s", o, msg)
	}
}

func TestPushH(t *testing.T) {
	dropCollection(cHits)

	p, _ := New(cfg, info)
	wg := sync.WaitGroup{}

	for i := 0; i < bufSize+1; i++ {
		wg.Add(1)
		go func() {
			p.(*ps).pushH(Hit{At: time.Now()})
			wg.Done()
		}()
	}

	wg.Wait()

	if len(p.(*ps).sH) > 1 {
		t.Error("ps.pushH failed: Buffered Hits didn't flushed. Buffer length:", len(p.(*ps).sH))
	}

	n := time.Now()
	h, err := p.Hits(n.Add(time.Millisecond), n.AddDate(0, 0, -1))
	if err != nil {
		t.Fatal("ps.Hits exits with error:", err)
	}

	if len(h) != bufSize+1 {
		t.Errorf("ps.pushH failed: Wrong number of Hits were pushed.\nGot : %d\nWant: %d", len(h), bufSize+1)
	}
}

func TestListen(t *testing.T) {
	f := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	p, _ := New(cfg, info)
	s := httptest.NewServer(p.Listen(f))
	c := &http.Client{}

	agn := "Mongo-Test"
	req, _ := http.NewRequest("HEAD", s.URL, nil)
	req.Header.Set("User-Agent", agn)
	req.Header.Set("Referer", agn)

	n := time.Now()
	r, err := c.Do(req)
	if err != nil {
		t.Fatal("http.Head exits with error:", err)
	}
	defer r.Body.Close()

	h, ok := p.(*ps).sH[0].(Hit)
	if !ok {
		t.Fatal("Listen failed: A ")
	}

	if h.At.Sub(n) > time.Millisecond ||
		h.Method != "HEAD" ||
		h.Path != "/" ||
		h.Status != http.StatusTeapot ||
		h.Dur == 0 ||
		h.From[:len("127.0.0.1")] != "127.0.0.1" ||
		h.Referer != agn ||
		h.Agent != agn {
		t.Error("ps.Listen failed: got ", h)
	}
}

func TestHits(t *testing.T) {
	dropCollection(cHits)

	now := time.Now()
	spl := []Hit{
		Hit{At: now.Add(-time.Hour), Method: "GET"}, // the time out of filtered range
		Hit{At: now.Add(-time.Second), Method: "POST"},
		Hit{At: now.Add(-time.Minute), Method: "PUT"},
	}

	pu, _ := New(cfg, info)
	p := pu.(*ps)

	for _, h := range spl {
		p.pushH(h)
	}

	// test expired time
	_, err := p.Hits(now.Add(-p.exp), now.Add(-p.exp).Add(-time.Hour))
	if err == nil {
		t.Error("ps.Hits failed: expired time input should return error")
	}

	s, err := p.Hits(now, now.Add(-30*time.Minute))
	if err != nil {
		t.Error("ps.Hits failed:", err)
	}

	if len(s) != len(spl)-1 {
		t.Errorf("ps.Hits failed: Returned slice length %d, want: %d", len(s), len(spl)-1)
	}
	if len(s) == 0 ||
		s[0].Method != "POST" ||
		now.Add(-time.Second).Sub(s[0].At) > time.Second/10 {
		t.Error("ps.Hits failed. Got:", s[0])
	}
}
