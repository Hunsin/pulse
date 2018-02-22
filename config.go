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

type Config struct {
	BackUpDir    string
	Expire       time.Duration
	ErrorHandler func(error, ...interface{})
	LevelFilter  func([]byte) string
}

func openFile(d, c string) (*os.File, error) {
	return os.OpenFile(fmt.Sprintf("%s/%s.log", d, c), os.O_APPEND|os.O_CREATE, 0666)
}

func defaultHandler(dir string) (func(error, ...interface{}), error) {
	if err := os.MkdirAll(dir, os.ModeDir|0666); err != nil {
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

		if _, ok := v[0].(Beat); !ok {
			o = h
		}
		for i := range v {
			fmt.Fprintln(o, v[i])
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fmt.Fprintln(b, n, err)
		}
	}, nil
}

func defaultFilter(b []byte) string {
	return ""
}
