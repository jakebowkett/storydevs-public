package service

import (
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
)

func populateOwnerPersProfile(tx sd.Tx, r sd.Resource) error {
	tx.Get(r, `
		SELECT
			slug as PersProfile
		FROM
			profile
		WHERE
			profile.ref_id = $1
	`, r.OwnerId())
	return nil
}

func personaOwnsResource(tx sd.Tx, tblName, slug string, persId int64) error {
	q := fmt.Sprintf(`
		FROM
			personas,
			%s
		WHERE
			(
				%s.slug = $1 AND
				%s.ref_id = $2
			) OR (
				personas.id = $2 AND
				personas.admin = true
			)`,
		tblName,
		tblName,
		tblName,
	)
	exists, err := tx.Exists(q, slug, persId)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("persona does not own resource it is trying to modify")
	}
	return nil
}

type tmpSpan struct {
	P    int
	Kind string
	Text string
	Link sd.NullString
	B    sd.NullBool
	I    sd.NullBool
	U    sd.NullBool
}

func retrieveSpans(tx sd.Tx, tblName string, refId int64) (sd.RichText, error) {

	var ss []tmpSpan
	err := tx.Select(&ss, fmt.Sprintf(`
		SELECT
			p,
			kind,
			text,
			link,
			b,
			i,
			u
		FROM
			%s_span
		WHERE
			ref_id = $1
		ORDER BY
			span ASC`, tblName),
		refId,
	)
	if err != nil {
		return nil, tx.Rollback(err)
	}

	var rt sd.RichText
	pIdx := -1
	for _, s := range ss {
		if pIdx != s.P {
			pIdx++
			rt = append(rt, sd.Paragraph{Kind: s.Kind})
		}
		var ff []string
		if s.B.Bool {
			ff = append(ff, "b")
		}
		if s.I.Bool {
			ff = append(ff, "i")
		}
		if s.U.Bool {
			ff = append(ff, "u")
		}
		rt[pIdx].Span = append(rt[pIdx].Span, sd.Span{
			Text:   s.Text,
			Link:   s.Link,
			Format: ff,
		})
	}

	return rt, nil
}

func length(s string) int {
	return len([]rune(s))
}

func in(ss []string, s string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}

func genSlug(title, id string) string {
	letterOrNumber := regexp.MustCompile(`[^\pL|\d]+`)
	title = letterOrNumber.ReplaceAllString(title, "-")
	title = strings.Trim(title, "-")
	title += "-"
	title += id
	return title
}

func genId(length int) string {

	// Seed with the current time or it'll
	// generate the same ID every time.
	rand.Seed(time.Now().Unix())

	// We don't use hyphens in the charset because
	// those are used to separate words in the slug.
	base64 := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_="
	set := strings.Split(base64, "")
	s := ""
	for i := 0; i < length; i++ {
		s += set[rand.Intn(len(set))]
	}
	return s
}

// This is not intended to be a fool-proof, spec-abiding
// URL checker. It is intended to find likely errors that
// could result from mis-typing or incomplete copy-pastes.
// It is not to be depended upon in a security context.
func validUrl(s string) bool {
	re := regexp.MustCompile(
		`^(http(s)?://)?` + // scheme is optional to allow e.g. "storydevs.com"
			`[\pL\d]+` + // must start with at least one letter or number
			`(\.[\pL\d]+)*` + // allow for sub-domains
			`\.[\pL]{2,}` + // top-level domain must be two or more letters
			`(/[^/?#]+)*` + // any number of path segments that don't contain query/fragment
			`/?` + // optional trailing slash
			`(\?(([^&#]+=[^&#]*)((&|;)[^&#]+=[^&#]*)*)?)?` + // optional query
			`(#.*)?$`) // optional fragment
	return re.MatchString(s)
}

type argCount int

func (ac *argCount) Next() string {
	*ac++
	return "$" + strconv.Itoa(int(*ac))
}

func buildExistsOr(arg *argCount, alias string, n int, names ...string) string {
	return varBuildExistsOr("id", arg, alias, n, names...)
}

func varBuildExistsOr(id string, arg *argCount, alias string, n int, names ...string) string {

	root := names[0]
	tbl := names[1]
	name := names[1]
	if len(names) > 2 {
		name = names[2]
	}
	refTbl := root
	if alias != "" {
		refTbl = alias
	}

	var where []string
	for i := 0; i < n; i++ {
		where = append(where, fmt.Sprintf("tbl.%s = %s", name, arg.Next()))
	}

	return fmt.Sprintf(`
		EXISTS (
			SELECT
				1
			FROM
				%s_%s AS tbl
			WHERE
				tbl.ref_id = %s.%s AND
				(%s)
		)
		`, root, tbl, refTbl, id, strings.Join(where, " OR "),
	)
}

/*
	Insert new files' names into table 'file' to allow
	checking privacy settings on user-associated files.
*/
func addFiles(tx sd.Tx, tbl *sd.DbTable, id int64, kind string) error {
	q := fmt.Sprintf(`INSERT INTO file (%s, file) VALUES ($1, $2)`, kind)
	for _, f := range tbl.Written {
		f = filepath.Base(f)
		if _, err := tx.Exec(q, id, f); err != nil {
			return err
		}
	}
	return nil
}

/*
Note that adding files must be done last or they'll be
added then imemdiately removed.
*/
func updateFiles(tx sd.Tx, tbl *sd.DbTable, id int64, kind string) ([]string, error) {

	// Get a list of all the files that won't be retained.
	var toRemove []string
	var where []string
	args := []interface{}{id}
	cmp := `file.file != $%d`
	for i, retain := range tbl.Retain {

		// Build thumb filename.
		ext := filepath.Ext(retain)
		thumb := strings.TrimSuffix(retain, ext) + "_thumb" + ext

		/*
			We multiple i by 2 because we are adding two file
			names per iteration (the original file and thumb).
			We then add 2 because of zero indexing and the id
			being $1.
		*/
		n := (i * 2) + 2
		where = append(where, fmt.Sprintf(cmp, n))
		where = append(where, fmt.Sprintf(cmp, n+1))
		args = append(args, retain)
		args = append(args, thumb)

	}
	q := fmt.Sprintf(`
 		SELECT
			file.file
		FROM
			file
		WHERE
			file.%s = $1`,
		kind)
	if len(where) > 0 {
		q += ` AND (` + strings.Join(where, " AND ") + `)`
	}
	if err := tx.Select(&toRemove, q, args...); err != nil {
		return nil, err
	}

	// If there's nothing to remove return here.
	if toRemove == nil {
		// Rollback is done from the caller, not here.
		if err := addFiles(tx, tbl, id, kind); err != nil {
			return nil, err
		}
		return nil, nil
	}

	/*
		Delete the entries of those we don't need and return
		them so we can mirror the deletion on disk.
	*/
	where = nil
	args = []interface{}{id}
	cmp = "file.file = $%d"
	for i, del := range toRemove {
		where = append(where, fmt.Sprintf(cmp, i+2))
		args = append(args, del)
	}
	q = fmt.Sprintf(`
		DELETE FROM
			file
		WHERE
			file.%s = $1
	`, kind)
	q += " AND (" + strings.Join(where, " OR ") + ")"
	if _, err := tx.Exec(q, args...); err != nil {
		return nil, err
	}

	// Rollback is done from the caller, not here.
	if err := addFiles(tx, tbl, id, kind); err != nil {
		return nil, err
	}

	return toRemove, nil
}

/*
insertTables recursively inserts each table in the tree
tbl and returns the DB generated id of the root table.
It returns early upon encountering an error. It never
attempts to rollback or commit the transaction.

The refId argument should be the id of the persona the
resource belongs to.
*/
func insertTables(tx sd.Tx, tbl *sd.DbTable, refId int64, update bool) (rId int64, err error) {

	// Do this here because loop below uses len(tbl.Values)
	tbl.SafeAdd("ref_id", refId)

	/*
		When updating we want to preserve these columns
		in the root resource table.
	*/
	if update {
		omit := []string{"id", "ref_id", "slug", "created"}
		var newCols []string
		var newVals []interface{}
		for i, col := range tbl.Columns {
			if in(omit, col) {
				continue
			}
			newCols = append(newCols, col)
			newVals = append(newVals, tbl.Values[i])
		}
		tbl.Columns = newCols
		tbl.Values = newVals
	}

	/*
		We add one to i to offset it from zero as
		PostgreSQL counts value placeholders from
		1 not 0.
	*/
	vals := make([]string, len(tbl.Values))
	for i := range tbl.Values {
		vals[i] = "$" + strconv.Itoa(i+1)
	}

	var q string
	if update {
		tbl.Values = append(tbl.Values, tbl.Slug)
		q = fmt.Sprintf(
			"UPDATE %s SET (%s) = (%s) WHERE slug = $%d",
			tbl.Name,
			strings.Join(tbl.Columns, ", "),
			strings.Join(vals, ", "),
			len(tbl.Values),
		)
	} else {
		q = fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			tbl.Name,
			strings.Join(tbl.Columns, ", "),
			strings.Join(vals, ", "),
		)
	}

	// Leaf nodes don't need to return an id.
	if len(tbl.Tables) > 0 {
		err = tx.Get(&refId, q+" RETURNING id", tbl.Values...)
	} else {
		_, err = tx.Exec(q, tbl.Values...)
	}
	if err != nil {
		return refId, err
	}

	for _, t := range tbl.Tables {
		if _, err = insertTables(tx, t, refId, false); err != nil {
			return refId, err
		}
	}

	return refId, nil
}
