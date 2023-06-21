package setup

import (
	"io"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler/modal"
	"github.com/jakebowkett/storydevs/postgres"
	"github.com/jakebowkett/storydevs/postgres/service"
)

type multiCloser []io.Closer

func (mc multiCloser) Close() error {
	for _, c := range mc {
		c.Close()
	}
	return nil
}

func Dependencies(path string) (dep *sd.Dependencies, openHandles io.Closer) {

	c := mustConfig(path)
	log, logFile := logger(c)

	pw := password(c)
	em := mustEmailer(c)
	tryTx := mustTryer("tx", c, postgres.Retry)
	tryEm := mustTryer("email", c, sd.RetryDefault)
	tryDisk := mustTryer("disk", c, sd.RetryDisk)

	db := dbConnect(c)
	dbTypes(c, db)
	dbTables(c, db)

	cache := mustCache(c)
	h := mustHyphenator(c)
	vd := mustViewData(c, cache, h)
	mp := mapResources(vd)
	tmpl := templates(c, h)
	fu := fieldUpdaters()
	tm := timeMapping()

	dep = &sd.Dependencies{
		Config:          c,
		Logger:          log,
		Password:        pw,
		Emailer:         em,
		TryerTx:         tryTx,
		TryerEmail:      tryEm,
		TryerDisk:       tryDisk,
		Db:              db,
		Cache:           cache,
		Hyphenator:      h,
		ViewData:        vd,
		ResourceMapping: mp,
		TimeMapping:     tm,
		Templates:       tmpl,
		FieldUpdaters:   fu,
	}

	as := &service.Account{Dependencies: dep}
	ms := &modal.Service{Dependencies: dep}
	rs := sd.Resources{
		"settings": service.Settings{Dependencies: dep},
		"talent":   service.Talent{Dependencies: dep},
		"library":  service.Thread{Dependencies: dep, Mode: "library"},
		"forums":   service.Thread{Dependencies: dep, Mode: "forums"},
		"event":    service.Event{Dependencies: dep},
	}

	dep.Accounts = as
	dep.Resources = rs
	dep.Modals = ms

	firstAccount(c, log, db, as)

	return dep, multiCloser{
		logFile,
		db,
	}
}

func fieldUpdaters() map[string]sd.FieldUpdateFunc {
	fu := make(map[string]sd.FieldUpdateFunc)
	fu["InitTimezone"] = initTimezone
	return fu
}
