package service

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	sd "github.com/jakebowkett/storydevs"
)

type Talent struct {
	*sd.Dependencies
}

func (ts Talent) Create(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, error) {

	p, ok := r.(*sd.Profile)
	if !ok {
		return nil, errors.New("supplied resource is not of type *sd.Profile")
	}
	if p == nil {
		return nil, fmt.Errorf("supplied talent profile is nil")
	}

	var rId int64
	errs, err := ts.TryerTx.Try(func() error {
		r.SetSlug(tbl.AddSlug())
		tx, err := ts.Db.Begin()
		if err != nil {
			return err
		}
		if rId, err = insertTables(tx, tbl, tbl.Refs, false); err != nil {
			return tx.Rollback(err)
		}
		if err = addFiles(tx, tbl, rId, "profile"); err != nil {
			return tx.Rollback(err)
		}
		return tx.Commit()
	})

	log := ts.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, p.PersId).
			Data(sd.LK_PersHandle, p.PersHandle)
		return nil, errors.New("Unable to create talent profile.")
	}

	log.Info(reqId, "Created talent profile.").
		Data(sd.LK_ProfileId, rId).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	/*
		Since we're not retrieving projects from
		the database they have to be sorted here.
	*/
	sortProjects(p.Project)

	return nil, nil
}

func (ts Talent) Update(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, []string, error) {

	p, ok := r.(*sd.Profile)
	if !ok {
		return nil, nil, errors.New("supplied resource is not of type *sd.Profile")
	}
	if p == nil {
		return nil, nil, fmt.Errorf("supplied talent profile is nil")
	}

	slug := r.GetSlug()
	var rId int64
	var toRemove []string

	errs, err := ts.TryerTx.Try(func() error {

		tx, err := ts.Db.Begin()
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

		err = tx.Get(&p.Created, `
			SELECT
				created
			FROM
				profile
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
		if toRemove, err = updateFiles(tx, tbl, rId, "profile"); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := ts.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, p.PersId).
			Data(sd.LK_PersHandle, p.PersHandle).
			Data(sd.LK_ProfileSlug, slug)
		return nil, nil, fmt.Errorf("Unable to update talent profile.")
	}

	log.Info(reqId, "Updated talent profile.").
		Data(sd.LK_ProfileId, rId).
		Data(sd.LK_ProfileSlug, slug).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	/*
		Since we're not retrieving projects from
		the database they have to be sorted here.
	*/
	sortProjects(p.Project)

	return nil, toRemove, nil
}

func (ts Talent) Retrieve(reqId, slug string, o sd.ResOpts) (sd.Resource, error) {

	var p sd.Profile

	var private string
	if !o.GetPrivate {
		private = "profile.visibility != 'private' AND"
	}

	errs, err := ts.TryerTx.Try(func() error {

		tx, err := ts.Db.BeginRead()
		if err != nil {
			return err
		}

		err = tx.Get(&p, fmt.Sprintf(`
			SELECT
				personas.id     AS persId,
				personas.name   AS persName,
				personas.handle AS persHandle,
				
				profile.id,
				profile.slug,
				profile.created,
				profile.updated,
				
				profile.available,
				profile.visibility,
				
				profile.name,
				profile.summary,
				
				profile.website,
				profile.email,
				profile.discord
			FROM
				profile,
				personas
			WHERE
				%s
				personas.id = profile.ref_id AND
				profile.slug = $1`, private),
			slug,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		// No need to do a query here since we already have the slug.
		p.PersProfile = p.Slug

		err = tx.Select(&p.PersPronouns, `
			SELECT
				pronoun
			FROM
				pronouns
			WHERE
				ref_id=$1
		`, p.PersId)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Get(&p.Duration, `
			SELECT
				duration_start AS start,
				duration_end AS end
			FROM
				profile
			WHERE
				slug = $1`,
			slug,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&p.Compensation, `
			SELECT
				compensation
			FROM
				profile_compensation
			WHERE
				ref_id = $1`,
			p.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&p.Medium, `
			SELECT
				medium
			FROM
				profile_medium
			WHERE
				ref_id = $1`,
			p.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&p.Language, `
			SELECT
				language
			FROM
				profile_language
			WHERE
				ref_id = $1`,
			p.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Select(&p.Tag, `
			SELECT
				tag
			FROM
				profile_tag
			WHERE
				ref_id = $1`,
			p.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		type pjt struct {
			Id int64
			sd.Project
		}
		var pjts []pjt
		err = tx.Select(&pjts, `
			SELECT
				id,
				name,
				link,
				teamname,
				teamlink,
				start,
				finish
			FROM
				profile_project
			WHERE
				ref_id = $1
			ORDER BY
				finish DESC`,
			p.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		for _, pjt := range pjts {

			type role struct {
				Id int64
				sd.Role
			}
			var roles []role
			err = tx.Select(&roles, `
				SELECT
					id,
					name,
					comment
				FROM
					profile_project_role
				WHERE
					ref_id = $1`,
				pjt.Id,
			)
			if err != nil {
				return tx.Rollback(err)
			}

			for _, role := range roles {

				err = tx.Select(&role.Skill, `
					SELECT
						skill
					FROM
						profile_project_role_skill
					WHERE
						ref_id = $1`,
					role.Id,
				)
				if err != nil {
					return tx.Rollback(err)
				}

				err = tx.Select(&role.Duty, `
					SELECT
						duty
					FROM
						profile_project_role_duty
					WHERE
						ref_id = $1`,
					role.Id,
				)
				if err != nil {
					return tx.Rollback(err)
				}

				pjt.Role = append(pjt.Role, role.Role)
			}

			p.Project = append(p.Project, pjt.Project)
		}

		type ad struct {
			Id int64
			sd.Advertised
		}
		var ads []ad
		err = tx.Select(&ads, `
			SELECT
				id,
				skill
			FROM
				profile_advertised
			WHERE
				ref_id = $1`,
			p.Id,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		for _, ad := range ads {

			type ex struct {
				Id int64
				sd.Media
			}
			var exs []ex
			err = tx.Select(&exs, `
				SELECT
					id,
					alttext,
					title,
					project,
					info,
					kind,
					format,
					aspect,
					filename AS file
				FROM
					profile_advertised_example
				WHERE
					ref_id = $1`,
				ad.Id,
			)
			if err != nil {
				return tx.Rollback(err)
			}

			for _, ex := range exs {
				ad.Example = append(ad.Example, ex.Media)
			}

			p.Advertised = append(p.Advertised, ad.Advertised)
		}

		return tx.Commit()
	})

	log := ts.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_ProfileSlug, slug)
		return nil, err
	}

	log.Info(reqId, "Retrieved talent profile.").
		Data(sd.LK_ProfileSlug, slug).
		Data(sd.LK_ProfileId, p.Id).
		Data(sd.LK_ProfileName, p.Name.String).
		Data(sd.LK_PersId, p.PersId).
		Data(sd.LK_PersHandle, p.PersHandle)

	return &p, nil
}

func (ts Talent) Filter(reqId string, admin bool, filter map[string][]string) ([]sd.Resource, error) {

	var where []string
	var args []interface{}
	arg := new(argCount)
	from := make(map[string]bool)

	q := `
		SELECT
			profile.slug
		FROM
			profile
		`

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
		where = append(where, "profile.ref_id = personas.id")
		where = append(where, "("+strings.Join(w, " OR ")+")")
	}

	if vv, ok := filter["visibility"]; ok {
		var w []string
		for _, v := range vv {
			w = append(w, "profile.visibility = "+arg.Next())
			args = append(args, v)
		}
		where = append(where, "("+strings.Join(w, " OR ")+")")
	}

	if vv, ok := filter["persona"]; ok {
		where = append(where, "profile.ref_id = "+arg.Next())
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

	if _, ok := filter["available"]; ok {
		where = append(where, "profile.available = true")
	}

	if vv, ok := filter["duration"]; ok {

		if len(vv) != 2 {
			return nil, errors.New("expected duration to have a start and end value")
		}

		start := vv[0]
		end := vv[1]

		w := fmt.Sprintf(`
			(
				(
					%s <= profile.duration_start AND
					%s >= profile.duration_end
				) OR (
					%s >= profile.duration_start AND
					%s <= profile.duration_end
				) OR (
					%s <= profile.duration_end AND
					%s >= profile.duration_start
				)
			)`,
			arg.Next(), arg.Next(),
			arg.Next(), arg.Next(),
			arg.Next(), arg.Next(),
		)

		where = append(where, w)
		args = append(args,
			start, end,
			start, start,
			end, end,
		)
	}

	if vv, ok := filter["compensation"]; ok {
		where = append(where, buildExistsOr(arg, "", len(vv), "profile", "compensation"))
		for _, v := range vv {
			args = append(args, v)
		}
	}

	if vv, ok := filter["medium"]; ok {
		where = append(where, buildExistsOr(arg, "", len(vv), "profile", "medium"))
		for _, v := range vv {
			args = append(args, v)
		}
	}

	if vv, ok := filter["skill"]; ok {
		where = append(where, buildExistsOr(arg, "", len(vv), "profile", "advertised", "skill"))
		for _, v := range vv {
			args = append(args, v)
		}
	}

	if vv, ok := filter["project"]; ok {
		where = append(where, `(
			SELECT
				COUNT(*)
			FROM
				profile_project
			WHERE
				profile_project.ref_id = profile.id
		) >= `+arg.Next())
		if len(vv) != 1 {
			return nil, errors.New("expected exactly one argument for projects")
		}
		v := vv[0]
		if len(v) == 0 {
			return nil, errors.New("expected project string to be longer")
		}
		v = string(v[0])
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		if n < 1 || n > 4 {
			return nil, errors.New("num of projects out of expected range")
		}
		args = append(args, n)
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

	var slugs []string
	errs, err := ts.TryerTx.Try(func() error {
		return ts.Db.Select(&slugs, q, args...)
	})

	log := ts.Logger

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
		r, err := ts.Retrieve(reqId, slug, sd.ResOpts{
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

func (ts Talent) Delete(reqId, slug string, persId int64) (sd.Feedback, []string, error) {

	q := `DELETE FROM profile WHERE slug = $1`
	qFiles := `
		SELECT
			file.file
		FROM
			file,
			profile
		WHERE
			profile.slug = $1 AND
			file.profile = profile.id`

	var toRemove []string

	errs, err := ts.TryerTx.Try(func() error {

		tx, err := ts.Db.Begin()
		if err != nil {
			return err
		}

		// Check resource is actually owned by account.
		if err = personaOwnsResource(tx, "profile", slug, persId); err != nil {
			return tx.Rollback(err)
		}

		// Get file names associated with profile.
		if err = tx.Select(&toRemove, qFiles, slug); err != nil {
			return tx.Rollback(err)
		}

		if _, err := tx.Exec(q, slug); err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := ts.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_ProfileSlug, slug)
		return nil, nil, err
	}

	log.Info(reqId, "Deleted talent profile.").
		Data(sd.LK_ProfileSlug, slug)

	return nil, toRemove, nil
}

/*
Note: we are deliberately placing i ahead of j if the
former is larger because we're sorting projects from
most to least recently finished.
*/
func sortProjects(projects []sd.Project) {
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Finish > projects[j].Finish
	})
}
