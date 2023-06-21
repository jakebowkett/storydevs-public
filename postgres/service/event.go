package service

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
)

const (
	second = 1
	minute = second * 60
	hour   = minute * 60
	day    = hour * 24
	week   = day * 7
)

type Event struct {
	*sd.Dependencies
}

func (es Event) Create(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, error) {

	e, ok := r.(*sd.Event)
	if !ok {
		return nil, errors.New("supplied resource is not of type *sd.Event")
	}
	if e == nil {
		return nil, fmt.Errorf("supplied event is nil")
	}
	if err := dateTimeToUTC(e, tbl); err != nil {
		return nil, err
	}

	var rId int64
	errs, err := es.TryerTx.Try(func() error {
		r.SetSlug(tbl.AddSlug())
		tx, err := es.Db.Begin()
		if err != nil {
			return err
		}
		if rId, err = insertTables(tx, tbl, tbl.Refs, false); err != nil {
			return tx.Rollback(err)
		}
		if err = addFiles(tx, tbl, rId, "event"); err != nil {
			return tx.Rollback(err)
		}
		return tx.Commit()
	})

	log := es.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, e.PersId).
			Data(sd.LK_PersHandle, e.PersHandle)
		return nil, errors.New("unable to create event")
	}

	log.Info(reqId, "Created event.").
		Data(sd.LK_EventId, rId).
		Data(sd.LK_PersId, e.PersId).
		Data(sd.LK_PersHandle, e.PersHandle)

	return nil, nil
}

func (es Event) Update(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, []string, error) {

	e, ok := r.(*sd.Event)
	if !ok {
		return nil, nil, errors.New("supplied resource is not of type *sd.Event")
	}
	if e == nil {
		return nil, nil, fmt.Errorf("supplied event is nil")
	}
	if err := dateTimeToUTC(e, tbl); err != nil {
		return nil, nil, err
	}

	slug := r.GetSlug()
	var rId int64
	var toRemove []string

	errs, err := es.TryerTx.Try(func() error {

		tx, err := es.Db.Begin()
		if err != nil {
			return err
		}

		// Check resource is actually owned by account.
		if err = personaOwnsResource(tx, tbl.Name, slug, r.OwnerId()); err != nil {
			return tx.Rollback(err)
		}

		/*
			Delete all rows in tables directly referencing
			the root table where the resource id is the same.
			Other tables referencing these rows have ON DELETE
			CASCADE set and will handle the rest.
		*/
		q := fmt.Sprintf(`SELECT id FROM %s WHERE slug = $1`, tbl.Name)
		if err = tx.Get(&rId, q, slug); err != nil {
			return tx.Rollback(err)
		}
		for _, t := range tbl.Tables {
			q := fmt.Sprintf(`DELETE FROM %s WHERE ref_id = $1`, t.Name)
			_, err := tx.Exec(q, rId)
			if err != nil {
				return tx.Rollback(err)
			}
		}

		if _, err = insertTables(tx, tbl, tbl.Refs, true); err != nil {
			return tx.Rollback(err)
		}

		/*
			Delete entries in the 'file' table that reference
			files the resource no longer needs and collect their
			paths so we can return them for deletion on disk if
			this update is successful. Also add any new files as
			entries in 'file' table.
		*/
		if toRemove, err = updateFiles(tx, tbl, rId, "event"); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := es.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, e.PersId).
			Data(sd.LK_PersHandle, e.PersHandle).
			Data(sd.LK_EventSlug, slug)
		return nil, nil, fmt.Errorf("unable to update event")
	}

	log.Info(reqId, "Updated event.").
		Data(sd.LK_EventId, rId).
		Data(sd.LK_EventSlug, slug).
		Data(sd.LK_PersId, e.PersId).
		Data(sd.LK_PersHandle, e.PersHandle)

	return nil, toRemove, nil
}

func (es Event) Retrieve(reqId, slug string, o sd.ResOpts) (sd.Resource, error) {

	var e sd.Event

	var private string
	if !o.GetPrivate {
		private = "event.visibility != 'private' AND"
	}

	errs, err := es.TryerTx.Try(func() error {

		tx, err := es.Db.BeginRead()
		if err != nil {
			return err
		}

		err = tx.Get(&e, fmt.Sprintf(`
			SELECT
				personas.id     AS persId,
				personas.name   AS persName,
				personas.handle AS persHandle,

				event.id,
				event.slug,
				event.created,
				event.updated,

				event.name,
				event.summary,
				event.visibility,

				event.timezone,
				event.weekly,
				event.start,
				event.finish

			FROM
				event,
				personas
			WHERE
				%s
				personas.id = event.ref_id AND
				event.slug = $1`, private),
			slug,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		if err := populateOwnerPersProfile(tx, &e); err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&e.Tag, `
			SELECT
				tag
			FROM
				event_tag
			WHERE
				ref_id = $1`,
			e.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&e.Category, `
			SELECT
				category
			FROM
				event_category
			WHERE
				ref_id = $1`,
			e.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&e.Setting, `
			SELECT
				setting
			FROM
				event_setting
			WHERE
				ref_id = $1`,
			e.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		/*
			Transaction is rolled back inside
			retrieveSpans if there's an error.
		*/
		rt, err := retrieveSpans(tx, "event", e.Id)
		if err != nil {
			return err
		}
		e.Body = rt

		return tx.Commit()
	})

	log := es.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_EventSlug, slug)
		return nil, err
	}

	/*
		We do this part first while the dates are
		still in UTC time.
	*/
	now := time.Now().Unix()
	if e.Weekly.Bool {
		var end int64
		s := e.Start.DateTime
		f := e.Finish.DateTime
		if e.Finish.Null {
			end = s
		} else {
			end = f
		}
		if now > end {
			s, f, err := nextOccurrence(s, f, e.Timezone)
			if err != nil {
				return nil, err
			}
			e.Start.DateTime = s
			if !e.Finish.Null {
				e.Finish.DateTime = f
			}
		}
	}

	/*
		We add the offset to DateTime structs as they are examined
		in isolation when populating the UI; the additional context
		of the Event struct and its Timezone field is not available.
	*/
	off := 0
	if !in(sd.TzSkip, e.Timezone) {
		tz, err := time.LoadLocation(e.Timezone)
		if err != nil {
			return nil, err
		}
		_, off = time.Now().In(tz).Zone()
	}
	e.Start.TZOff = off
	if !e.Finish.Null {
		e.Finish.TZOff = off
	}

	log.Info(reqId, "Retrieved event.").
		Data(sd.LK_EventSlug, slug).
		Data(sd.LK_EventId, e.Id).
		Data(sd.LK_EventName, e.Name.String).
		Data(sd.LK_PersId, e.PersId).
		Data(sd.LK_PersHandle, e.PersHandle)

	return &e, nil
}

func nextOccurrence(s, f int64, tz string) (sAdj, fAdj int64, err error) {
	dates := [2]int64{s, f}
	now := time.Now()
	today := now.Weekday()
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return 0, 0, err
	}
	for i, d := range dates {
		t := time.Unix(d, 0).In(loc)
		day := t.Weekday()
		delta := int(day - today)
		if delta < 0 {
			delta += 7
		}
		next := time.Date(
			now.Year(),
			now.Month(),
			now.Day()+delta,
			t.Hour(),
			t.Minute(),
			0,
			0,
			time.UTC,
		)
		dates[i] = next.Unix()
	}
	return dates[0], dates[1], nil
}

/*
Take search range date and subtract the user's timezone
offset at that date (not the present moment).
*/
func filterDateTimeToUTC(filter map[string][]string, which string) (int64, error) {
	vv := filter[which]
	if len(vv) == 0 {
		return 0, fmt.Errorf(`filter key %q has no associated values`, which)
	}
	var n int64
	var err error
	if vv[0] == "Present" {
		n = time.Now().Unix()
	} else {
		n, err = strconv.ParseInt(vv[0], 10, 64)
		if err != nil {
			return 0, err
		}
	}
	tzName, ok := filter["timezone"]
	if !ok {
		return n, nil
	}
	if len(tzName) == 0 {
		return n, nil
	}
	off, err := offFromUTC(tzName[0], n)
	if err != nil {
		return 0, err
	}
	return n - int64(off), nil
}

func (es Event) Filter(reqId string, admin bool, filter map[string][]string) ([]sd.Resource, error) {

	var where []string
	var args []interface{}
	now := time.Now().Unix()
	arg := new(argCount)
	from := make(map[string]bool)

	q := `
		WITH v (now, wk) AS (
			VALUES(
				` + strconv.FormatInt(now, 10) + `::BIGINT,
				60 * 60 * 24 * 7
			)
		)
		SELECT
			e.slug
		FROM
			v,
			event AS e
		`

	/*
		Easier to deal with multiple return values
		if err is already declared.
	*/
	var err error
	overlaps := false
	offClient := 0

	if len(filter) == 0 {
		goto skip
	}

	if vv, ok := filter["persona_visibility"]; ok {
		from["personas"] = true
		var w []string
		for _, v := range vv {
			w = append(w, "personas.visibility = "+arg.Next())
			args = append(args, v)
		}
		where = append(where, "e.ref_id = personas.id")
		where = append(where, "("+strings.Join(w, " OR ")+")")
	}

	if vv, ok := filter["visibility"]; ok {
		var w []string
		for _, v := range vv {
			w = append(w, "e.visibility = "+arg.Next())
			args = append(args, v)
		}
		where = append(where, "("+strings.Join(w, " OR ")+")")
	}

	if vv, ok := filter["persona"]; ok {
		where = append(where, "e.ref_id = "+arg.Next())
		if len(vv) < 1 {
			return nil, errors.New("expected persona id while filtering talent db")
		}
		persId, err := strconv.ParseInt(vv[0], 10, 64)
		if err != nil {
			return nil, err
		}
		args = append(args, persId)
		goto skip
	}

	if vv, ok := filter["overlap"]; ok {
		overlaps = vv[0] == "overlap"
	}

	if tzName, ok := filter["timezone"]; ok {
		offClient, err = offFromUTC(tzName[0], now)
		if err != nil {
			return nil, err
		}
	}

	/*
		We open a scope here because the goto cannot skip
		over newly declared variables between the goto and
		its named label.
	*/
	{
		_, okS := filter["start"]
		_, okF := filter["finish"]
		s := int64(math.MinInt64)
		f := int64(math.MaxInt64)
		var err error
		if okS {
			s, err = filterDateTimeToUTC(filter, "start")
			if err != nil {
				return nil, err
			}
		}
		if okF {
			f, err = filterDateTimeToUTC(filter, "finish")
			if err != nil {
				return nil, err
			}
			/*
				We add one day to make the end of the range inclusive.
				This is only done if there is a finish range supplied
				otherwise we'd wrap the default value of max int.
			*/
			f += day
		}

		if overlaps {
			qMaker := func(opening string, offset int) string {
				off := ""
				if offset > 0 {
					off = " - " + strconv.Itoa(offset)
				}
				s := `
(
	/*
		Events set to local timezone have the user's timezone
		offset subtracted from event start/finish.
	*/
	` + opening + ` AND
	(
		e.start ` + off + ` >= %s AND e.start ` + off + ` < %s
	)
	OR
	(
		 e.finish IS NOT NULL AND
	 	(
	 		(e.finish ` + off + ` >= %s AND e.finish ` + off + `  < %s) OR
	 		(e.start  ` + off + `  < %s AND e.finish ` + off + ` >= %s)
	 	)
 	)
 	OR
 	(
 		-- For recurring events.
		e.weekly AND
 		(
 			(e.finish IS NOT NULL AND now > e.finish ` + off + `) OR
 			(e.finish IS NULL     AND now > e.start  ` + off + `)
		)
		AND
		(
	 		(now + (wk - ((now - (e.start ` + off + `)) %% wk))  >= %s AND
	 		 now + (wk - ((now - (e.start ` + off + `)) %% wk))   < %s) OR

	 		(e.finish IS NOT NULL AND
			 now + (wk - ((now - (e.finish ` + off + `)) %% wk)) >= %s AND
	 		 now + (wk - ((now - (e.finish ` + off + `)) %% wk))  < %s)
 		)
 	)
)`
				return fmt.Sprintf(s,
					arg.Next(), arg.Next(),
					arg.Next(), arg.Next(),
					arg.Next(), arg.Next(),
					arg.Next(), arg.Next(),
					arg.Next(), arg.Next())
			}
			var ss []string
			ss = append(ss, qMaker(`e.timezone  = 'local'`, offClient))
			ss = append(ss, qMaker(`e.timezone != 'local'`, 0))
			where = append(where, "("+strings.Join(ss, " OR ")+")")
			args = append(args,
				s, f, s, f,
				s, f, s, f,
				s, f, s, f,
				s, f, s, f,
				s, f, s, f)
		} else {
			qMaker := func(opening string, offset int) string {
				off := ""
				if offset > 0 {
					off = " - " + strconv.Itoa(offset)
				}
				s := `
(
	` + opening + `AND
	(
		e.start ` + off + ` >= %s AND
	 	(
		 	(e.finish ` + off + ` IS NULL AND e.start ` + off + ` < %s)
	 		OR
		 	(e.finish ` + off + ` IS NOT NULL AND e.finish ` + off + ` < %s)
		)
	)
	OR
	(
 		-- For recurring events.
		e.weekly AND
		now + (wk - ((now - e.start ` + off + `) %% wk)) >= %s AND
 		(
			(e.finish IS NOT NULL AND
			 now > e.finish ` + off + ` AND
			 now + (wk - ((now - e.finish ` + off + `) %% wk)) < %s) OR

			(e.finish IS NULL AND
			 now > e.start ` + off + ` AND
			 now + (wk - ((now - e.start ` + off + `) %% wk)) < %s)
		)
	)
)`
				return fmt.Sprintf(s,
					arg.Next(), arg.Next(), arg.Next(),
					arg.Next(), arg.Next(), arg.Next(),
				)
			}
			var ss []string
			ss = append(ss, qMaker(`e.timezone  = 'local'`, offClient))
			ss = append(ss, qMaker(`e.timezone != 'local'`, 0))
			where = append(where, "("+strings.Join(ss, " OR ")+")")
			args = append(args,
				s, f, f,
				s, f, f,
				s, f, f,
				s, f, f)
		}
	}

	if vv, ok := filter["time"]; ok {

		instCount, err := strconv.Atoi(vv[0])
		if err != nil {
			return nil, err
		}

		var or []string

		for i := 0; i < instCount; i++ {

			var and []string

			if vv, ok := filter[fmt.Sprintf("time.%d.day_start", i)]; ok {
				dayStart := vv[0]
				var nn []int
				if vv, ok := filter[fmt.Sprintf("time.%d.day_finish", i)]; ok {
					dayFinish := vv[0]
					nn = weekdayRangeFrom(dayStart, dayFinish, offClient)
				} else {
					nn = weekdayRangeFrom(dayStart, dayStart, offClient)
				}
				and = append(and, buildDayRangeOr(arg, len(nn), offClient, overlaps))
				for _, n := range nn {
					if overlaps {
						args = append(args,
							n, n+day,
							n, n+day,

							n, n+day,
							n, n+day)
					} else {
						args = append(args,
							n, n+day, n+day,
							n, n+day, n+day)
					}
				}
			}

			vvS, okS := filter[fmt.Sprintf("time.%d.start", i)]
			vvF, okF := filter[fmt.Sprintf("time.%d.finish", i)]
			var hrS int
			var hrF int
			if okS {
				hrS, err = strconv.Atoi(vvS[0])
				if err != nil {
					return nil, err
				}
			}
			if okF {
				hrF, err = strconv.Atoi(vvF[0])
				if err != nil {
					return nil, err
				}
			} else {
				hrF = day
			}
			if okS || okF {
				// hrS, hrF = hourRangeFrom(hrS, hrF, offClient)
				if overlaps {
					qMaker := func(opening string, offset int) string {
						off := ""
						if offset > 0 {
							off = " - " + strconv.Itoa(offset)
						}
						s := `
(
	` + opening + ` AND
	(
		(
			(e.start  + %d ` + off + `) %% %d >= %s AND
			(e.start  + %d ` + off + `) %% %d  < %s)
		OR
		(
		 	 e.finish IS NOT NULL AND
		 	(e.finish + %d ` + off + `) %% %d  > %s AND
		 	(e.finish + %d ` + off + `) %% %d <= %s
		)
	)
)`
						return fmt.Sprintf(s,
							offClient, day, arg.Next(),
							offClient, day, arg.Next(),
							offClient, day, arg.Next(),
							offClient, day, arg.Next(),
						)
					}
					var ss []string
					ss = append(ss, qMaker(`e.timezone  = 'local'`, offClient))
					ss = append(ss, qMaker(`e.timezone != 'local'`, 0))
					and = append(and, "("+strings.Join(ss, " OR ")+")")
					args = append(args,
						hrS, hrF,
						hrS, hrF,
						hrS, hrF,
						hrS, hrF,
					)
				} else {
					qMaker := func(opening string, offset int) string {
						off := ""
						if offset > 0 {
							off = " - " + strconv.Itoa(offset)
						}
						s := `
(
	` + opening + `AND
	(e.start  + %d ` + off + `) %% %d >= %s AND
	(
		(
			 e.finish IS NULL AND
			(e.start  + %d  ` + off + `) %% %d <= %s
		)
		OR
		(
			 e.finish IS NOT NULL AND
		 	(e.finish + %d  ` + off + `) %% %d <= %s
		)
	)
)`
						return fmt.Sprintf(s,
							offClient, day, arg.Next(),
							offClient, day, arg.Next(),
							offClient, day, arg.Next())
					}
					var ss []string
					ss = append(ss, qMaker(`e.timezone  = 'local'`, offClient))
					ss = append(ss, qMaker(`e.timezone != 'local'`, 0))
					and = append(and, "("+strings.Join(ss, " OR ")+")")
					args = append(args,
						hrS, hrF, hrF,
						hrS, hrF, hrF)
				}
			}

			if len(and) > 0 {
				or = append(or, "("+strings.Join(and, " AND ")+")")
			}
		}

		if len(or) > 0 {
			where = append(where, "("+strings.Join(or, " OR ")+")")
		}
	}

	if vv, ok := filter["category"]; ok {
		where = append(where, buildExistsOr(arg, "e", len(vv), "event", "category"))
		for _, v := range vv {
			args = append(args, v)
		}
	}

	if vv, ok := filter["setting"]; ok {
		where = append(where, buildExistsOr(arg, "e", len(vv), "event", "setting"))
		for _, v := range vv {
			args = append(args, v)
		}
	}

skip:

	if len(from) > 0 {
		for tbl := range from {
			q += ",\n" + tbl + "\n"
		}
	}

	if len(where) > 0 {
		q += "WHERE " + strings.Join(where, " AND \n")
	}

	q += " ORDER BY start ASC"

	var slugs []string
	errs, err := es.TryerTx.Try(func() error {
		tx, err := es.Db.Begin()
		if err != nil {
			return err
		}
		if err := tx.Select(&slugs, q, args...); err != nil {
			return tx.Rollback(err)
		}
		return tx.Commit()
	})

	log := es.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs))
		return nil, err
	}

	if len(slugs) == 0 {
		return nil, nil
	}

	var rr []sd.Resource
	for _, slug := range slugs {
		r, err := es.Retrieve(reqId, slug, sd.ResOpts{
			GetPrivate:       true,
			CalledFromFilter: true,
		})
		if err != nil {
			return nil, err
		}
		if e := r.(*sd.Event); e.Timezone == "local" {
			e.Start.DateTime -= int64(offClient)
			if !e.Finish.Null {
				e.Finish.DateTime -= int64(offClient)
			}
		}
		rr = append(rr, r)
	}

	return rr, nil
}

func (es Event) Delete(reqId, slug string, persId int64) (sd.Feedback, []string, error) {

	q := `DELETE FROM event WHERE slug = $1`
	qFiles := `
		SELECT
			file.file
		FROM
			file,
			event
		WHERE
			event.slug = $1 AND
			file.event = event.id`

	var toRemove []string

	errs, err := es.TryerTx.Try(func() error {

		tx, err := es.Db.Begin()
		if err != nil {
			return err
		}

		// Check resource is actually owned by account.
		if err = personaOwnsResource(tx, "event", slug, persId); err != nil {
			return tx.Rollback(err)
		}

		// Get file names associated with event.
		if err = tx.Select(&toRemove, qFiles, slug); err != nil {
			return tx.Rollback(err)
		}

		if _, err := tx.Exec(q, slug); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := es.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_EventSlug, slug)
		return nil, nil, err
	}

	log.Info(reqId, "Deleted event.").
		Data(sd.LK_EventSlug, slug)

	return nil, toRemove, nil
}

func offFromUTC(tzName string, at int64) (int, error) {
	if in(sd.TzSkip, tzName) {
		return 0, nil
	}
	tz, err := time.LoadLocation(tzName)
	if err != nil {
		return 0, err
	}
	_, off := time.Unix(at, 0).In(tz).Zone()
	return off, nil
}

func dateTimeToUTC(e *sd.Event, tbl *sd.DbTable) error {
	off, err := offFromUTC(e.Timezone, e.Start.DateTime)
	if err != nil {
		return err
	}
	tbl.SafeAdd("start", e.Start.DateTime-int64(off))
	if !e.Finish.Null {
		off, err := offFromUTC(e.Timezone, e.Finish.DateTime)
		if err != nil {
			return err
		}
		tbl.SafeAdd("finish", e.Finish.DateTime-int64(off))
	}
	return nil
}

func buildDayRangeOr(arg *argCount, n, offClient int, overlaps bool) string {
	var where []string
	if overlaps {
		qMaker := func(opening string, offset int) string {
			off := ""
			if offset > 0 {
				off = " - " + strconv.Itoa(offset)
			}
			s := `
(
	` + opening + ` AND
	(
		(
			(e.start  + %d ` + off + `) %% wk >= %s AND
			(e.start  + %d ` + off + `) %% wk  < %s
		)
		OR
		(
			e.finish IS NOT NULL AND
			(e.finish + %d ` + off + `) %% wk >= %s AND
			(e.finish + %d ` + off + `) %% wk  < %s
		)
	)
)`
			return fmt.Sprintf(s,
				offClient, arg.Next(),
				offClient, arg.Next(),
				offClient, arg.Next(),
				offClient, arg.Next(),
			)
		}
		for i := 0; i < n; i++ {
			where = append(where, qMaker(`e.timezone  = 'local'`, offClient))
			where = append(where, qMaker(`e.timezone != 'local'`, 0))
		}
	} else {
		qMaker := func(opening string, offset int) string {
			off := ""
			if offset > 0 {
				off = " - " + strconv.Itoa(offset)
			}
			s := `
(
	` + opening + ` AND
	(e.start + %d ` + off + `) %% wk >= %s AND
	(
	 	(e.finish IS NULL AND     (e.start  + %d ` + off + `) %% wk < %s) OR
	 	(e.finish IS NOT NULL AND (e.finish + %d ` + off + `) %% wk < %s)
	)
)`
			return fmt.Sprintf(s,
				offClient, arg.Next(),
				offClient, arg.Next(),
				offClient, arg.Next(),
			)
		}
		for i := 0; i < n; i++ {
			where = append(where, qMaker(`e.timezone  = 'local'`, offClient))
			where = append(where, qMaker(`e.timezone != 'local'`, 0))
		}
	}
	return "(" + strings.Join(where, " OR ") + ")"
}

func weekdayRangeFrom(start, finish string, offClient int) (nn []int) {
	ns := indexOf(epochDayOrder, start)
	nf := indexOf(epochDayOrder, finish)
	if ns == -1 || nf == -1 {
		return nil
	}
	nf++
	if nf > ns {
		for i := ns; i < nf; i++ {
			nn = append(nn, i*day)
		}
	} else {
		for i := ns; i < len(epochDayOrder); i++ {
			nn = append(nn, i*day)
		}
		for i := 0; i < nf; i++ {
			nn = append(nn, i*day)
		}
	}
	return nn
}

func indexOf(ss []string, s string) int {
	for i := range ss {
		if ss[i] == s {
			return i
		}
	}
	return -1
}

var epochDayOrder = []string{
	"thu", // January 1st, 1970 is a Thursday
	"fri",
	"sat",
	"sun",
	"mon",
	"tue",
	"wed",
}

func printQueryWithArgs(q string, args []interface{}) {
	for i, a := range args {
		q = strings.Replace(
			q,
			"$"+strconv.Itoa(i+1),
			fmt.Sprintf("%v", a),
			1,
		)
	}
	println(q)
}
