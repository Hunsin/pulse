package pulse

import (
	"fmt"
	"os"
	"time"
)

const (
	cBeats  = "pulse_beats"
	cHits   = "pulse_hits"
	bufSize = 1 << 8
)

// A Config is a configuration for initializing a Pulse.
type Config struct {

	// BackUpDir is the path to the directory where storing log files.
	// Beats and Hits will be written to individual file if the connection
	// to database is lost. It is used only if no ErrorHandler is applied.
	BackUpDir string

	// Expire specifies the validity time of Beats and Hits. They will be
	// deleted once expired. If it's 0, the data will persist forever.
	Expire time.Duration

	// ErrorHandler is a function that takes an error and a slice of Beats
	// or Hits. Customize it overrides the default handler, which writes
	// data in BackUpDir.
	ErrorHandler func(error, ...interface{})

	// LevelFilter reads the written bytes and returns the severity level.
	LevelFilter func([]byte) string
}

func openFile(d, c string) (*os.File, error) {
	if d[len(d)-1] == '/' {
		d = d[:len(d)-1]
	}

	return os.OpenFile(fmt.Sprintf("%s/%s.log", d, c), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
}

func defaultHandler(dir string) (func(error, ...interface{}), error) {
	if err := os.MkdirAll(dir, os.ModeDir|0775); err != nil {
		return nil, err
	}
	b, err := openFile(dir, cBeats)
	if err != nil {
		return nil, err
	}
	h, err := openFile(dir, cHits)
	if err != nil {
		b.Close()
		return nil, err
	}

	return func(err error, v ...interface{}) {
		n := time.Now()
		o := b

		if _, ok := v[0].(Hit); ok {
			o = h
		}
		for i := range v {
			fmt.Fprint(o, v[i])
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, n, err)
			fmt.Fprint(b, Beat{"", n, "ERROR", err.Error()})
		}
	}, nil
}

func defaultFilter(b []byte) string {
	return ""
}
