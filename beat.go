package pulse

import (
	"time"

	"github.com/globalsign/mgo/bson"
)

type Beat struct {
	id    bson.ObjectId `bson:"_id,omitempty"`
	At    time.Time
	Level string
	Body  string
}

// inserB flushes the buffered beats into MongoDB.
func (p *ps) insertB() {
	p.wg.Add(1)

	if len(p.sB) == 0 {
		p.wg.Done()
		return
	}

	b := make([]interface{}, len(p.sB))
	copy(b, p.sB)
	p.sB = []interface{}{}

	go func() {
		if err := p.db.C(cBeats).Insert(p.sB...); err != nil {
			p.eh(err, b...)
		}

		p.wg.Done()
	}()
}

// pushB appends b to beats buffer.
func (p *ps) pushB(b Beat) {
	p.mB.Lock()
	defer p.mB.Unlock()

	p.sB = append(p.sB, b)
	if len(p.sB) > bufSize {
		p.insertB()
	}
}

// Beats returns a list of Beats which issued between t and u.
func (p *ps) Beats(t, u time.Time) ([]Beat, error) {
	if t.After(u) {
		t, u = u, t
	}
	if p.exp != 0 && time.Since(u) > p.exp {
		return nil, ErrExpired
	}
	p.Flush()

	var bs []Beat
	return bs, p.db.C(cBeats).Find(
		bson.M{
			"at": bson.M{
				"$gte": t,
				"$lt":  u,
			},
		}).Sort("-at").All(&bs)
}

// Write implements the io.Writer interface.
func (p *ps) Write(b []byte) (int, error) {
	n := time.Now()                    // get time earlier
	if l := len(b) - 1; b[l] == '\n' { // trim newline
		b = b[:l]
	}

	p.pushB(Beat{At: n, Level: p.lf(b), Body: string(b)})
	return len(b), nil
}
