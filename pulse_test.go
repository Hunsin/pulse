package pulse

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/globalsign/mgo"
)

var (
	cfg  = Config{BackUpDir: "./log", Expire: 48 * time.Hour}
	info = &mgo.DialInfo{Addrs: []string{""}, Timeout: 8 * time.Second}
)

func init() {
	flag.StringVar(&info.Addrs[0], "addr", "localhost:27017", "pulse test database address")
	flag.StringVar(&info.Database, "db", "pulse", "pulse test database name")
	flag.StringVar(&info.Username, "user", "", "pulse test database user")
	flag.StringVar(&info.Password, "pwd", "", "pulse test database password")
}

func dropCollection(c string) error {
	return client.(*ps).db.C(c).DropCollection()
}

func TestMain(m *testing.M) {
	flag.Parse()
	Start(cfg, info)
	c := m.Run()

	if p, ok := client.(*ps); ok {
		p.db.DropDatabase()
	}
	os.Exit(c)
}

func TestNew(t *testing.T) {
	p, err := New(cfg, info)
	if err != nil {
		t.Fatal("New failed:", err)
	}

	// check TTL of collections
	for _, c := range []string{cBeats, cHits} {
		idx, _ := p.(*ps).db.C(c).Indexes()

		var i int
		for i = 0; i < len(idx); i++ {
			if idx[i].Key[0] == "at" &&
				idx[i].ExpireAfter == cfg.Expire {
				break
			}
		}

		if i == len(idx) {
			t.Errorf("New failed: Index of collection %s not set", c)
		}
	}

	// check if error handler and level filter had been created
	if p.(*ps).eh == nil || p.(*ps).lf == nil {
		t.Error("New failed: default error handler or level filter func wasn't initialized")
	}
}

func TestStartAndClient(t *testing.T) {
	p, _ := Client()
	if p == nil {
		t.Fatal("Start or Client failed: Can not access default Pulse")
	}
}
