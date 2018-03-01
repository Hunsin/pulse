package pulse

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
	"time"
)

var (
	// pt is "pattern of timestamp"
	pt     = "[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9.]* [+-][0-9]{4}"
	reg    = regexp.MustCompile(pt + " [A-Z]{4,5} [A-Za-z ]*\n")
	regHit = regexp.MustCompile(pt + " GET / 200 1s 127.0.0.1 test go")
)

func TestDefaultErrorHandler(t *testing.T) {

	// removing existing file
	logBeat := fmt.Sprintf("%s/%s.log", cfg.BackUpDir, cBeats)
	logHit := fmt.Sprintf("%s/%s.log", cfg.BackUpDir, cHits)
	os.Remove(logBeat)
	os.Remove(logHit)

	// initial samples
	n := time.Now()
	b := Beat{At: n, Level: "INFO", Body: "Log message"}
	h := Hit{
		At:      n,
		Method:  "GET",
		Path:    "/",
		Status:  200,
		Dur:     time.Second,
		From:    "127.0.0.1",
		Referer: "test",
		Agent:   "go",
	}
	e := errors.New("My custom error")

	fn, err := defaultHandler(cfg.BackUpDir)
	if err != nil {
		t.Fatal("defaultHandler failed:", err)
	}

	// capture STDERR
	stderr := os.Stderr
	r, w, _ := os.Pipe()
	buf := &bytes.Buffer{}
	os.Stderr = w
	go func() {
		io.Copy(buf, r)
	}()

	fn(e, b)

	// test Beat log file
	out, err := ioutil.ReadFile(logBeat)
	if err != nil {
		t.Fatal("defaultHandler failed: Beat log file was not created")
	}

	m := reg.FindAllSubmatch(out, -1)
	if len(m) != 2 {
		t.Errorf("defaultHandler failed. log Beat: %s", out)
	}

	fn(e, h)

	// test Hit log file
	out, err = ioutil.ReadFile(logHit)
	if err != nil {
		t.Fatal("defaultHandler failed: Hit log file was not created")
	}

	if !regHit.Match(out) {
		t.Errorf("defaultHandler failed. log Beat: %s", out)
	}

	// test if writing logBeat when writing logHit
	out, _ = ioutil.ReadFile(logBeat)
	m = reg.FindAllSubmatch(out, -1)
	if len(m) != 3 {
		t.Error("defaultHandler failed. Beat log file wasn't written when writing Hit")
	}

	// test STDERR
	out = buf.Bytes()
	m = reg.FindAllSubmatch(out, -1)
	if len(m) != 2 {
		t.Errorf("defaultHandler failed. log STDERR: %s", out)
	}

	// restore STDERR
	os.Stderr = stderr
}
