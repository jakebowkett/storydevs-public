package service

import (
	"fmt"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
)

type Settings struct {
	*sd.Dependencies
}

/*
These aren't used for account settings. Users cannot create/delete settings.
*/
func (s Settings) Create(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, error) {
	return nil, nil
}
func (s Settings) Delete(reqId, slug string, persId int64) (sd.Feedback, []string, error) {
	return nil, nil, nil
}

func (s Settings) Filter(reqId string, admin bool, filter map[string][]string) ([]sd.Resource, error) {
	var rr []sd.Resource
	for _, f := range s.ViewData.Mode["settings"].Editor {
		base := sd.ResourceBase{Slug: f.Name}
		rr = append(rr, &sd.Settings{ResourceBase: base, Field: f})
	}
	return rr, nil
}

/*
Make a copy of settings so we don't modify
the original. Then manually populate it.
*/
func (s Settings) Retrieve(reqId, slug string, o sd.ResOpts) (sd.Resource, error) {
	return nil, nil
}

func (s Settings) Update(reqId string, r sd.Resource, tbl *sd.DbTable) (sd.Feedback, []string, error) {

	fb := make(sd.Feedback)
	m := tblToMap(tbl)
	pId := r.OwnerId()
	cat := r.GetSlug()
	var toRemove []string
	handle := r.GetHandle()

	errs, err := s.TryerTx.Try(func() error {

		tx, err := s.Db.Begin()
		if err != nil {
			return err
		}

		switch cat {
		case "identity":

			var avatar interface{}
			if a, ok := m["avatar"]; ok {
				avatar = a
			} else {
				avatar = nil
			}

			handle = m["handle"].(string)

			/*
				For an explanation of the redeemed condition here
				see the comment in the handleAndEmailFree method.
			*/
			exists, err := tx.Exists(`
				FROM
					reserved
				WHERE
					handle ILIKE $1 AND
					code IS NULL AND
					redeemed IS NULL`,
				handle)
			if err != nil {
				return tx.Rollback(err)
			}
			if exists {
				fb.Add("handle", "handle is already in use")
				return tx.Rollback(nil)
			}

			exists, err = tx.Exists(`
				FROM
					accounts,
					personas
				WHERE
					accounts.id = personas.acc_id AND
					personas.id != $2 AND
					personas.handle ILIKE $1 AND
					accounts.deleted IS NULL AND
					accounts.code IS NULL`,
				handle, pId)
			if err != nil {
				return tx.Rollback(err)
			}
			if exists {
				fb.Add("handle", "handle is already in use")
				return tx.Rollback(nil)
			}

			// Update persona table.
			_, err = tx.Exec(`
				UPDATE personas SET (
					handle,
					name,
					updated,
					avatar
				) = (
					$1, $2, $3, $4
				)
				WHERE
					id = $5
				`,
				handle,
				m["name"],
				time.Now().Unix(),
				avatar,
				pId,
			)
			if err != nil {
				return tx.Rollback(err)
			}

			// Find pronouns table.
			var pronouns []interface{}
			for _, t := range tbl.Tables {
				if strings.HasSuffix(t.Name, "pronouns") {
					pronouns = append(pronouns, t.Values[0])
				}
			}

			// Remove all pronoun instances.
			_, err = tx.Exec(`DELETE FROM pronouns WHERE ref_id = $1`, pId)
			if err != nil {
				return tx.Rollback(err)
			}

			// Insert new ones.
			for _, pn := range pronouns {
				_, err = tx.Exec(`
					INSERT INTO pronouns (
						ref_id,
						pronoun
					) VALUES (
						$1, $2
					)`,
					r.OwnerId(),
					pn,
				)
				if err != nil {
					return tx.Rollback(err)
				}
			}

			/*
				Delete entries in the 'file' table that reference
				files the resource no longer needs and collect their
				paths so we can return them for deletion on disk if
				this update is successful. Also add any new files as
				entries in 'file' table.
			*/
			if toRemove, err = updateFiles(tx, tbl, pId, "persona"); err != nil {
				return tx.Rollback(err)
			}

		case "privacy":

			_, err := s.Db.Exec(`
				UPDATE
					personas
				SET
					visibility = $1
				WHERE
					id = $2`,
				m["visibility"],
				pId,
			)
			if err != nil {
				return err
			}

		default:
			return tx.Rollback(fmt.Errorf("service: unknown settings category %q", cat))
		}

		return tx.Commit()
	})

	log := s.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_PersId, r.OwnerId()).
			Data(sd.LK_PersHandle, r.GetHandle()).
			Data(sd.LK_PersSlug, r.GetPersSlug())
		return nil, nil, fmt.Errorf("Unable to update persona.")
	}

	if len(fb) > 0 {
		return fb, nil, nil
	}

	log.Info(reqId, "Updated persona.").
		Data(sd.LK_PersId, r.OwnerId()).
		Data(sd.LK_PersHandle, handle).
		Data(sd.LK_PersSlug, r.GetPersSlug())

	return nil, toRemove, nil
}

func tblToMap(tbl *sd.DbTable) map[string]interface{} {
	m := make(map[string]interface{})
	for i, name := range tbl.Columns {
		m[name] = tbl.Values[i]
	}
	return m
}
