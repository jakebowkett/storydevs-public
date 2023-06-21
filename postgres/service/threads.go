package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
)

const fbThreadLocked = "Thread locked. No replies, edits, or deletions are possible."

type Thread struct {
	*sd.Dependencies
	Mode string
}

func (t Thread) Create(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, error) {

	p, ok := r.(*sd.Post)
	if !ok {
		return nil, errors.New("supplied resource is not of type *sd.Post")
	}
	if p == nil {
		return nil, fmt.Errorf("supplied %s post is nil", t.Mode)
	}

	tbl.Tables = append(tbl.Tables, &sd.DbTable{
		Name:    "post_kind",
		Columns: []string{"kind"},
		Values:  []interface{}{t.Mode},
	})

	if p.ThreadSlug == "" {
		return t.createThread(reqId, p, tbl)
	} else {
		return t.createReply(reqId, p, tbl)
	}
}

func (t Thread) createThread(reqId string, p *sd.Post, tbl *sd.DbTable) (sd.Feedback, error) {

	fb := make(sd.Feedback)
	tbl.SafeAdd("idx", 1)

	var tId int64
	var tSlug string
	errs, err := t.TryerTx.Try(func() error {

		tSlug = tbl.AddSlug()
		p.Slug = tSlug

		tx, err := t.Db.Begin()
		if err != nil {
			return err
		}

		if tId, err = insertTables(tx, tbl, tbl.Refs, false); err != nil {
			return tx.Rollback(err)
		}

		// Set OP's thread to its post ID.
		_, err = tx.Exec(`
			UPDATE
				post
			SET
				thread = id
			WHERE
				slug = $1`,
			tSlug)
		if err != nil {
			return tx.Rollback(err)
		}

		if err = addFiles(tx, tbl, tId, "post"); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := t.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, p.PersId).
			Data(sd.LK_PersHandle, p.PersHandle)
		return nil, fmt.Errorf("Unable to create %s thread.", t.Mode)
	}

	if len(fb) > 0 {
		return fb, nil
	}

	log.InfoF(reqId, "Created %s thread.", t.Mode).
		Data(sd.LK_ThreadId, tId).
		Data(sd.LK_ThreadSlug, tSlug).
		Data(sd.LK_ThreadTitle, p.Name.String).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	return nil, nil
}

func (t Thread) createReply(reqId string, p *sd.Post, tbl *sd.DbTable) (sd.Feedback, error) {

	fb := make(sd.Feedback)
	tmp := struct {
		Idx    int64
		Thread int64
		Slug   string
		Name   sd.NullString
		Locked sd.NullBool
	}{}

	var rId int64
	var rSlug string
	errs, err := t.TryerTx.Try(func() error {

		rSlug = tbl.AddSlug()
		p.Slug = rSlug

		tx, err := t.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&tmp, `
			SELECT
				idx,
				thread
			FROM
				post
			WHERE
				thread = (
					SELECT
						id
					FROM
						post
					WHERE
						slug=$1
				)
			ORDER BY
				idx DESC`,
			p.ThreadSlug)
		if err != nil {
			return err
		}
		err = tx.Get(&tmp, `
			SELECT
				locked,
				slug,
				name
			FROM
				post
			WHERE
				id = $1`,
			tmp.Thread)
		if err != nil {
			return err
		}
		if tmp.Locked.Bool && !p.Admin.Bool {
			fb.Add("general", fbThreadLocked)
			return tx.Rollback(nil)
		}
		tmp.Idx++
		tbl.SafeAdd("thread", tmp.Thread)
		tbl.SafeAdd("idx", tmp.Idx)

		if rId, err = insertTables(tx, tbl, tbl.Refs, false); err != nil {
			return tx.Rollback(err)
		}
		if err = addFiles(tx, tbl, rId, "post"); err != nil {
			return tx.Rollback(err)
		}
		return tx.Commit()
	})

	log := t.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, p.PersId).
			Data(sd.LK_PersHandle, p.PersHandle)
		return nil, fmt.Errorf("Unable to create %s reply.", t.Mode)
	}

	if len(fb) > 0 {
		return fb, nil
	}

	log.InfoF(reqId, "Created %s reply.", t.Mode).
		Data(sd.LK_ThreadId, tmp.Thread).
		Data(sd.LK_ThreadSlug, tmp.Slug).
		Data(sd.LK_ThreadTitle, tmp.Name.String).
		Data(sd.LK_PostId, rId).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	return nil, nil
}

func (t Thread) Update(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, []string, error) {

	fb := make(sd.Feedback)
	p, ok := r.(*sd.Post)
	if !ok {
		return nil, nil, errors.New("supplied resource is not of type *sd.Post")
	}
	if p == nil {
		return nil, nil, fmt.Errorf("supplied %s post is nil", t.Mode)
	}

	var rId int64
	slug := r.GetSlug()
	thread := struct {
		Id   int64
		Slug string
		Name string
	}{}
	var toRemove []string

	errs, err := t.TryerTx.Try(func() error {

		tx, err := t.Db.Begin()
		if err != nil {
			return err
		}

		// Check resource is actually owned by account.
		if err = personaOwnsResource(tx, tbl.Name, slug, r.OwnerId()); err != nil {
			return tx.Rollback(err)
		}

		if !r.GetAdmin() {
			exists, err := tx.Exists(`
				FROM
					post
				WHERE
					locked = true AND
					(
						SELECT
							post.thread
						FROM
							post
						WHERE
							post.slug = $1
					) = post.id`,
				slug)
			if err != nil {
				return tx.Rollback(err)
			}
			if exists {
				fb.Add("general", fbThreadLocked)
				return tx.Rollback(nil)
			}
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

		err = tx.Get(&thread, `
			SELECT
				post.id,
				post.slug,
				post.name
			FROM
				post
			WHERE
				(
					SELECT
						post.thread
					FROM
						post
					WHERE
						post.slug = $1
				) = post.id`,
			slug)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Get(&p.Created, `
			SELECT
				created
			FROM
				post
			WHERE
				slug = $1`,
			slug)
		if err != nil {
			return tx.Rollback(err)
		}

		/*
			Delete entries in the 'file' table that reference
			files the resource no longer needs and collect their
			paths so we can return them for deletion on disk if
			this update is successful. Also add any new files as
			entries in 'file' table.
		*/
		if toRemove, err = updateFiles(tx, tbl, rId, "post"); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := t.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, p.PersId).
			Data(sd.LK_PersHandle, p.PersHandle).
			Data(sd.LK_PostSlug, slug)
		return nil, nil, fmt.Errorf("Unable to update %s post.", t.Mode)
	}

	if len(fb) > 0 {
		return fb, nil, nil
	}

	p.ThreadSlug = thread.Slug
	p.ThreadId = thread.Id

	// Add word count to post that's calculated when validating it.
	for i, name := range tbl.Columns {
		if name == "words" {
			p.Words = tbl.Values[i].(int)
		}
	}

	log.InfoF(reqId, "Updated %s post.", t.Mode).
		Data(sd.LK_ThreadId, thread.Id).
		Data(sd.LK_ThreadSlug, thread.Slug).
		Data(sd.LK_ThreadTitle, thread.Name).
		Data(sd.LK_PostId, rId).
		Data(sd.LK_PostSlug, slug).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	return nil, toRemove, nil
}

func (t Thread) RetrieveById(reqId string, id int64, o sd.ResOpts) (sd.Resource, error) {
	return t.retrieve(reqId, "id", id, o)
}

func (t Thread) Retrieve(reqId, slug string, o sd.ResOpts) (sd.Resource, error) {
	return t.retrieve(reqId, "slug", slug, o)
}

func (t Thread) retrieve(reqId, kind string, arg interface{}, o sd.ResOpts) (sd.Resource, error) {

	var p sd.Post
	var slug string
	var private string
	if !o.GetPrivate {
		private = "post.visibility != 'private' AND"
	}

	errs, err := t.TryerTx.Try(func() error {

		tx, err := t.Db.BeginRead()
		if err != nil {
			return err
		}

		switch kind {
		case "slug":
			slug = arg.(string)
		case "id":
			err := tx.Get(&slug, `
				SELECT
					slug
				FROM
					post
				WHERE
					id = $1`,
				arg)
			if err != nil {
				return tx.Rollback(err)
			}
		}

		isThreadRoot, err := tx.Exists(`
			FROM
				post
			WHERE
				slug = $1 AND
				id = thread
		`, slug)
		if err != nil {
			return err
		}

		var pp []sd.Post
		if isThreadRoot {
			pp, err = thread(tx, slug, private)
		} else {
			pp, err = post(tx, slug, private)
		}
		if err != nil {
			return tx.Rollback(err)
		}

		for i := range pp {

			err = tx.Select(&pp[i].PersPronouns, `
				SELECT
					pronoun
				FROM
 					pronouns
				WHERE
					ref_id = $1`,
				pp[i].PersId)
			if err != nil {
				return tx.Rollback(err)
			}

			/*
				Transaction is rolled back inside
				retrieveSpans if there's an error.
			*/
			rt, err := retrieveSpans(tx, "post", pp[i].Id)
			if err != nil {
				return err
			}
			pp[i].Body = rt

			err = tx.Select(&pp[i].Kind, `
				SELECT
					kind
				FROM
					post_kind
				WHERE
					ref_id = $1`,
				pp[i].Id)
			if err != nil {
				return tx.Rollback(err)
			}

			err = tx.Select(&pp[i].Category, `
				SELECT
					category
				FROM
					post_category
				WHERE
					ref_id = $1`,
				pp[i].Id)
			if err != nil {
				return tx.Rollback(err)
			}

			err = tx.Select(&pp[i].Tag, `
				SELECT
					tag
				FROM
					post_tag
				WHERE
					ref_id = $1`,
				pp[i].Id)
			if err != nil {
				return tx.Rollback(err)
			}
		}

		p = pp[0]
		p.Reply = pp[1:]

		return tx.Commit()
	})

	log := t.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_ThreadSlug, slug)
		return nil, err
	}

	var resourceKind string
	if p.IsReply() {
		resourceKind = "reply"
	} else {
		resourceKind = "thread"
	}

	log.InfoF(reqId, "Retrieved %s %s.", t.Mode, resourceKind).
		Data(sd.LK_ThreadSlug, slug).
		Data(sd.LK_ThreadId, p.Id).
		Data(sd.LK_ThreadTitle, p.Name.String).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	return &p, nil
}

func thread(tx sd.Tx, slug, private string) ([]sd.Post, error) {
	var pp []sd.Post
	err := tx.Select(&pp, fmt.Sprintf(`
		SELECT
			personas.id         AS persId,
			personas.handle     AS persHandle,
			personas.name       AS persName,
			personas.avatar     AS persAvatar,
			personas.visibility AS persVis,
			personas.admin,
			
			post.thread AS threadid,
			post.id,
			post.slug,
			post.created,
			post.updated,
			
			post.pinned,
			post.locked,
			post.deleted,
			post.visibility,
			post.name,
			post.summary,
			post.words
		FROM
			post,
			personas
		WHERE
			(
				SELECT
					id
				FROM
					post
				WHERE
					%s
					slug = $1
			) = post.thread AND
			personas.id = post.ref_id
		ORDER BY
			post.idx ASC`, private),
		slug,
	)
	return pp, err
}

func post(tx sd.Tx, slug, private string) ([]sd.Post, error) {
	var p sd.Post
	err := tx.Get(&p, fmt.Sprintf(`
		SELECT
			personas.id         AS persId,
			personas.handle     AS persHandle,
			personas.name       AS persName,
			personas.avatar     AS persAvatar,
			personas.visibility AS persVis,
			personas.admin,
			
			post.thread AS threadid,
			post.id,
			post.slug,
			post.created,
			post.updated,
			
			post.pinned,
			post.locked,
			post.deleted,
			post.visibility,
			post.name,
			post.summary,
			post.words
		FROM
			post,
			personas
		WHERE
			%s
			post.slug = $1 AND
			personas.id = post.ref_id`, private),
		slug,
	)
	if err != nil {
		return nil, err
	}
	tmp := struct {
		Name   string
		Slug   string
		Locked sd.NullBool
	}{}
	err = tx.Get(&tmp, `
		SELECT
			name,
			slug,
			locked
		FROM
			post
		WHERE
			id = (
				SELECT
					thread
				FROM
					post
				WHERE
					slug = $1
			)`,
		slug)
	if err != nil {
		return nil, err
	}
	p.Locked.Bool = tmp.Locked.Bool
	p.Name.String = "Re: " + tmp.Name
	p.ThreadSlug = tmp.Slug
	return []sd.Post{p}, nil
}

func (t Thread) Filter(reqId string, admin bool, filter map[string][]string) ([]sd.Resource, error) {

	var where []string
	var args []interface{}
	arg := new(argCount)

	if len(filter) == 0 {
		goto skip
	}

	if vv, ok := filter["deleted"]; ok {
		if len(vv) == 0 || len(vv) > 1 {
			return nil, fmt.Errorf("expected exactly 1 value for deleted while filtering %s", t.Mode)
		}
		if vv[0] == "false" {
			where = append(where, "post.deleted IS NULL")
		}
	}

	if vv, ok := filter["visibility"]; ok {
		for _, v := range vv {
			where = append(where, "post.visibility = "+arg.Next())
			args = append(args, v)
		}
	}

	if vv, ok := filter["persona"]; ok {
		where = append(where, "post.ref_id = "+arg.Next())
		if len(vv) < 1 {
			return nil, fmt.Errorf("expected persona id while filtering %s", t.Mode)
		}
		persId, err := strconv.ParseInt(vv[0], 10, 64)
		if err != nil {
			return nil, err
		}
		args = append(args, persId)
		goto skip
	}

	if vv, ok := filter["category"]; ok {
		where = append(where, varBuildExistsOr("thread", arg, "", len(vv), "post", "category"))
		for _, v := range vv {
			args = append(args, v)
		}
	}

skip:

	where = append(where, buildExistsOr(arg, "", 1, "post", "kind"))
	args = append(args, t.Mode)

	q := ""
	if _, ok := filter["thread"]; ok {

		/*
			The filter "thread" is incompatible with "deleted"
			and/or "persona_visibility". Thread IDs are obtained
			from the *last* reply, meaning any test for deletion
			or visibility will refer to the final reply, not the
			thread OP.
		*/
		_, okDeleted := filter["deleted"]
		_, okPersVis := filter["persona_visibility"]
		if okDeleted || okPersVis {
			return nil, fmt.Errorf(`incompatible use of filter "thread" with "deleted" and/or "persona_visibility" while filtering %s`, t.Mode)
		}

		q = `
			SELECT
				post.thread
			FROM
				/*
					Create a table containing the thread IDs and
					final post's idx in each of those threads.
				*/
				(
					SELECT
						post.thread,
						MAX(post.idx) AS idx
					FROM
						post%s
					GROUP BY
						post.thread
				) p1
			INNER JOIN
				/*
					Create a table containing thread OP IDs and
					their pinned status. (Only the OP of a thread
					is flagged as pinned/locked.)
				*/
				(
					SELECT
						post.id,
						post.thread,
						post.pinned
					FROM
						post
					WHERE
						post.thread = post.id
				) p2
				ON p1.thread = p2.thread
			INNER JOIN
				/*
					Join the post table back onto these sub-tables
					so that we have access to the _final_ post's
					created column. We don't want to sort by p2's
					created because it's the OP and p1 won't allow
					selecting another columns without featuring it
					in GROUP BY or an aggregate function.
				*/
				post
				ON
					post.idx = p1.idx AND
					post.thread = p1.thread
			WHERE
				%s
			ORDER BY
				p2.pinned ASC,
				post.created DESC
		`
		if admin {
			q = fmt.Sprintf(q, "", "%s")
		} else {
			// We filter out deleted/hidden posts for regular users.
			q = fmt.Sprintf(
				q,
				`,
						personas
					WHERE
						post.deleted IS NULL AND
						personas.id = post.ref_id AND
						personas.visibility != 'private'`,
				`%s`)
		}
	} else {
		q = `
			SELECT
				post.id
			FROM
				post
			WHERE
				%s
			ORDER BY
				post.pinned ASC,
				post.created DESC
		`
	}

	if len(where) > 0 {
		q = fmt.Sprintf(q, strings.Join(where, " AND \n"))
	}

	var ids []int64
	errs, err := t.TryerTx.Try(func() error {
		return t.Db.Select(&ids, q, args...)
	})

	log := t.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs))
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	var rr []sd.Resource
	for _, id := range ids {
		r, err := t.RetrieveById(reqId, id, sd.ResOpts{
			GetPrivate:       true,
			CalledFromFilter: true,
		})
		if err != nil {
			return nil, err
		}
		rr = append(rr, r)
	}

	return rr, nil
}

func (t Thread) Delete(reqId, slug string, persId int64) (sd.Feedback, []string, error) {

	fb := make(sd.Feedback)
	q := `UPDATE post SET deleted = true WHERE slug = $1`

	errs, err := t.TryerTx.Try(func() error {

		tx, err := t.Db.Begin()
		if err != nil {
			return err
		}

		// Check resource is actually owned by account.
		if err = personaOwnsResource(tx, "post", slug, persId); err != nil {
			return tx.Rollback(err)
		}

		exists, err := tx.Exists(`
			FROM
				personas,
				post
			WHERE
				locked = true AND
				(
					personas.id = $2 AND
					personas.admin IS NULL
				) AND (
					SELECT
						post.thread
					FROM
						post
					WHERE
						post.slug = $1
				) = post.id`,
			slug, persId)
		if err != nil {
			return tx.Rollback(err)
		}
		if exists {
			fb.Add("general", fbThreadLocked)
			return tx.Rollback(nil)
		}

		if _, err := tx.Exec(q, slug); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := t.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_ProfileSlug, slug)
		return nil, nil, err
	}

	if len(fb) > 0 {
		return fb, nil, nil
	}

	log.InfoF(reqId, "Flagged %s post as deleted.", t.Mode).
		Data(sd.LK_PostSlug, slug)

	return nil, nil, nil
}

type metaRecent struct {
	Category string
	Count    int
}

func metaByName(mr []metaRecent, name string) metaRecent {
	for i := range mr {
		if mr[i].Category == name {
			return mr[i]
		}
	}
	return metaRecent{}
}

func (t Thread) Meta(reqId, kind string, search sd.Fields) error {

	var mr []metaRecent

	minute := int64(60)
	hour := minute * 60
	day := hour * 24
	week := day * 7
	lastWeek := time.Now().Unix() - week

	err := t.Db.Select(&mr, `
		SELECT
			pc.category,
			COUNT(*)
		FROM
			post
		INNER JOIN
			post_category pc
		ON
			post.thread = pc.ref_id
		INNER JOIN
			personas
		ON
			post.ref_id = personas.id
		INNER JOIN
			post_kind k
		ON
			post.id = k.ref_id
		WHERE
			post.created > $1 AND
			post.deleted IS NULL AND
			k.kind = $2 AND
			personas.visibility = 'public'
		GROUP BY
			pc.category`,
		lastWeek, kind)
	if err != nil {
		return err
	}

	for i := range search {
		ff := search[i].Field
		for i := range ff {
			vv := ff[i].Value
			for i := range vv {
				n := metaByName(mr, vv[i].Name).Count
				if n > 0 {
					vv[i].Data = strconv.Itoa(n)
				}
			}
		}
	}

	return nil
}
