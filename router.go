package storydevs

import (
	"net/http"
	"sync"
	"time"
)

type Guard = func(r *Request) bool
type Handler = func(w http.ResponseWriter, r *Request)
type Vars = map[string]string
type Request = struct {
	Id      string
	Request *http.Request
	Vars    Vars
	User    interface{}
	Status  int
	Error   error
	Began   time.Time
}

type TimedSet interface {
	Add(string)
	Has(string) bool
}

type timedSet struct {
	delay time.Duration
	rwMu  sync.RWMutex
	set   map[string]int
}

/*
NewTimedSet returns a set that will remove any
values added to it after d time has passed. If
d is zero the value will never be removed.
*/
func NewTimedSet(d time.Duration) TimedSet {
	return &timedSet{
		delay: d,
		set:   make(map[string]int),
	}
}

func (ts *timedSet) Add(v string) {
	ts.rwMu.Lock()
	ts.set[v]++
	id := ts.set[v]
	ts.rwMu.Unlock()
	if ts.delay == 0 {
		return
	}
	go func() {
		time.Sleep(ts.delay)
		ts.rwMu.Lock()
		defer ts.rwMu.Unlock()
		/*
			Return if the Id has been incremented,
			indicating the value has been added again.
		*/
		if id != ts.set[v] {
			return
		}
		delete(ts.set, v)
	}()
}

func (ts *timedSet) Has(v string) bool {
	ts.rwMu.RLock()
	_, ok := ts.set[v]
	ts.rwMu.RUnlock()
	return ok
}

type IdGenerator interface {
	NewId() string
}

type Dependencies struct {
	Config          *Config
	Logger          Logger
	Password        Password
	Emailer         Emailer
	TryerTx         Tryer
	TryerEmail      Tryer
	TryerDisk       Tryer
	Db              DB
	Cache           Cache
	Hyphenator      Hyphenator
	ViewData        *ViewData
	ResourceMapping ResourceMapping
	TimeMapping     map[string]string
	Templates       View
	Accounts        Accounts
	Resources       Resources
	Modals          Modals
	FieldUpdaters   map[string]FieldUpdateFunc
}
