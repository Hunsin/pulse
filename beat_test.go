package pulse

import (
	"log"
	"sync"
	"testing"
	"time"
)

func TestPushB(t *testing.T) {
	dropCollection(cBeats)

	now := time.Now()
	msg := "Hello Mongo!"
	wg := sync.WaitGroup{}

	for i := 0; i < bufSize+1; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.(*ps).pushB(Beat{
				At:   now.Add(time.Duration(-i) * time.Second),
				Body: msg,
			})
		}()
	}

	wg.Wait()
	if len(client.(*ps).sB) > 1 {
		t.Error("ps.pushB failed: Buffered beats didn't flushed. Buffer length:", len(client.(*ps).sB))
	}

	s, err := client.Beats(time.Now(), now.AddDate(0, 0, -1))
	if err != nil {
		t.Fatal("ps.Beats exits with error:", err)
	}

	if len(s) != bufSize+1 {
		t.Errorf("ps.pushB failed: Wrong number of beats were pushed.\nGot : %d\nWant: %d", len(s), bufSize+1)
	}
}

func TestBeats(t *testing.T) {
	dropCollection(cBeats)

	msg := "Hello Mongo!"
	now := time.Now()
	spl := []Beat{
		Beat{At: now.AddDate(0, -1, 0), Level: "Info", Body: msg},
		Beat{At: now.AddDate(0, 0, -1), Level: "Debug", Body: msg},
		Beat{At: now, Level: "Warn", Body: msg},
		Beat{At: now.AddDate(0, -2, 0), Level: "Error", Body: msg}, // the time out of filtered range
	}

	pu, _ := New(cfg, info)
	p := pu.(*ps)

	for _, b := range spl {
		p.pushB(b)
	}

	// test expired time
	_, err := p.Beats(now.Add(-p.exp), now.Add(-p.exp).Add(-time.Hour))
	if err == nil {
		t.Error("ps.Beats failed: expired time input should return error")
	}

	// MongoDB timestamp resolution is 1ms,
	// so we add 1ms to make sure the data will be in filtered range
	s, err := p.Beats(now.Add(time.Millisecond), now.AddDate(0, -1, -5))
	if err != nil {
		t.Fatal("ps.Beats failed:", err)
	}

	if len(s) != len(spl)-1 {
		t.Errorf("ps.Beats failed: Returned slice length %d, want: %d", len(s), len(spl)-1)
	}

	if len(s) == 0 ||
		s[0].id.String() == "" ||
		s[0].Level != "Warn" ||
		s[0].Body != msg ||
		now.Sub(s[0].At) > time.Second/10 {
		t.Error("ps.Beats failed. Got:", s[0])
	}
}

func TestWrite(t *testing.T) {
	p := client.(*ps)
	p.sB = []interface{}{}

	m := "Hello World"
	n := time.Now()
	p.Write([]byte(m + "\n"))

	if len(p.sB) == 0 {
		t.Fatal("ps.Write failed: No Beat was pushed")
	}

	b := p.sB[0].(Beat)
	if b.At.Sub(n) > time.Millisecond {
		t.Errorf("ps.Write failed: Wrong timestamp\nGot : %s\nWant: %s", b.At, n)
	}
	if b.Body != m {
		t.Errorf("ps.Write failed: Wrong message body\nGot : %s\nWant: %s", b.Body, m)
	}
}

func BenchmarkWriteBeats(b *testing.B) {
	p, err := New(cfg, info)
	if err != nil {
		b.Fatal(err)
	}

	l := log.New(p, "", 0)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		l.Println(i)
	}

	p.Flush()
	b.StopTimer()
}
