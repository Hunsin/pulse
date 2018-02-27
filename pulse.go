package pulse

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/globalsign/mgo"
)

var (
	// ErrExpired is returned if an operation requests Beats or Hits within
	// the time already expired.
	ErrExpired = errors.New("pulse: requested time already expired")

	// ErrNoPulse is returned by Client() if the default Pulse is not created.
	ErrNoPulse = errors.New("pulse: No Pulse was initialized")

	// TimeFormat defines the layout of timestamp.
	TimeFormat = "2006-01-02 15:04:05.999999 -0700"

	client Pulse
)

// A Pulse represents an event.
type Pulse interface {

	// Write implements the io.Writer interface.
	Write([]byte) (int, error)

	// Beats returns a slice of Beats during the time interval.
	Beats(time.Time, time.Time) ([]Beat, error)

	// Listen is a http middleware which logs the request's information.
	Listen(http.Handler) http.Handler

	// Hits returns a slice of Hits during the time interval
	Hits(time.Time, time.Time) ([]Hit, error)

	// Flush pushes all buffered data into database.
	Flush()
}

// A ps implements the Pulse interface.
type ps struct {
	db     *mgo.Database
	sB, sH []interface{}               // slice of beat; slice of Hit
	mB, mH sync.Mutex                  // mutex of beat; mutex of Hit
	exp    time.Duration               // expired time
	eh     func(error, ...interface{}) // error handler
	lf     func([]byte) string         // level filter
	wg     sync.WaitGroup
}

func (p *ps) Flush() {
	p.wg.Add(1)

	// push []beat
	go func() {
		p.mB.Lock()
		defer p.mB.Unlock()

		p.insertB()
		p.wg.Done()
	}()

	// push []Hit
	p.mH.Lock()
	defer p.mH.Unlock()

	p.insertH()

	p.wg.Wait()
}

// New returns a new Pulse by given configurations.
func New(cfg Config, info *mgo.DialInfo) (Pulse, error) {
	if cfg.BackUpDir == "" && cfg.ErrorHandler == nil {
		return nil, errors.New("pulse: either BackUpDir or ErrorHandler must specified")
	}

	s, err := mgo.DialWithInfo(info)
	if err != nil {
		return nil, err
	}
	db := s.DB("")

	// seting "time to live" index
	if cfg.Expire != 0 {
		ttl := mgo.Index{
			Key:         []string{"at"},
			ExpireAfter: cfg.Expire,
		}

		if err = db.C(cBeats).EnsureIndex(ttl); err != nil {
			return nil, err
		}
		if err = db.C(cHits).EnsureIndex(ttl); err != nil {
			return nil, err
		}
	}

	p := &ps{
		db:  db,
		exp: cfg.Expire,
		eh:  cfg.ErrorHandler,
		lf:  cfg.LevelFilter,
	}
	if p.eh == nil {
		p.eh, err = defaultHandler(cfg.BackUpDir)
		if err != nil {
			return nil, err
		}
	}
	if p.lf == nil {
		p.lf = defaultFilter
	}

	return p, nil
}

// Start acts just like New() but keeps the Pulse as default so it can
// be shared between different packages. Retrieve the value by Client().
func Start(cfg Config, info *mgo.DialInfo) (Pulse, error) {
	p, err := New(cfg, info)
	if err != nil {
		return nil, err
	}

	client = p
	return p, nil
}

// Client returns the default Pulse. If default was not set, an error
// is returned.
func Client() (Pulse, error) {
	if client == nil {
		return nil, ErrNoPulse
	}
	return client, nil
}
