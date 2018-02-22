package pulse

import (
	"net/http"
	"time"

	"github.com/globalsign/mgo/bson"
)

type Hit struct {
	id      bson.ObjectId `bson:"_id,omitempty"`
	At      time.Time     `bson:""`
	Method  string        `bson:""`
	Path    string        `bson:""`
	Status  int           `bson:""`
	Dur     time.Duration `bson:""`
	From    string        `bson:""`
	Referer string        `bson:""`
	Agent   string        `bson:""`
}

// A httpRecorder implements http.ResponseWriter interface.
// It records response's status code.
type httpRecorder struct {
	w http.ResponseWriter
	c int
}

func (w *httpRecorder) Header() http.Header {
	return w.w.Header()
}

func (w *httpRecorder) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

func (w *httpRecorder) WriteHeader(code int) {
	w.w.WriteHeader(code)
	w.c = code
}

// insertH flushes the buffered Hits into MongoDB.
func (p *ps) insertH() {
	p.wg.Add(1)

	if len(p.sH) == 0 {
		p.wg.Done()
		return
	}

	h := make([]interface{}, len(p.sH))
	copy(h, p.sH)
	p.sH = []interface{}{}

	go func() {
		if err := p.db.C(cHits).Insert(h...); err != nil {
			p.eh(err, h...)
		}

		p.wg.Done()
	}()
}

// pushH appends h to Hits buffer.
func (p *ps) pushH(h Hit) {
	p.mH.Lock()
	defer p.mH.Unlock()

	p.sH = append(p.sH, h)
	if len(p.sH) > bufSize {
		p.insertH()
	}
}

// Hits returns a list of Hits which issued between t and u.
func (p *ps) Hits(t, u time.Time) ([]Hit, error) {
	if t.After(u) {
		t, u = u, t
	}
	if p.exp != 0 && time.Since(u) > p.exp {
		return nil, ErrExpired
	}
	p.Flush()

	var hs []Hit
	return hs, p.db.C(cHits).Find(
		bson.M{
			"at": bson.M{
				"$gte": t,
				"$lt":  u,
			},
		}).Sort("-at").All(&hs)
}

func (p *ps) Listen(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		s := &httpRecorder{w: w}
		h.ServeHTTP(s, r)

		p.pushH(Hit{
			At:      t,
			Method:  r.Method,
			Path:    r.URL.Path,
			Status:  s.c,
			Dur:     time.Since(t),
			From:    r.RemoteAddr,
			Referer: r.Referer(),
			Agent:   r.UserAgent(),
		})
	})
}
