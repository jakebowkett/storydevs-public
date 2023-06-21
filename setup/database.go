package setup

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/postgres"
)

func dbConnect(c *sd.Config) sd.DB {
	return postgres.MustConnect(c.Credentials.DbConn)
}

func dbTypes(c *sd.Config, db sd.DB) {

	f, err := ioutil.ReadFile(c.PathDbTypes)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	for _, s := range strings.SplitAfter(string(f), ";") {

		/*
			The first fields of type will be: "CREATE",
			"TYPE", "[type_name]" . . . and so on. We
			get the index of "CREATE" and truncate the
			string to its start to remove any proceeding
			comments. Then split the string into fields
			and grab the name and values.
		*/
		idx := strings.Index(s, "CREATE")
		if idx < 0 {
			continue
		}
		ss := strings.Fields(s[idx:])
		typeName := ss[2]
		vv := ss[6 : len(ss)-1]

		// Clean up the values.
		for i, v := range vv {
			vv[i] = strings.Trim(v, ",'")
		}

		// Grab the values associated with type name.
		var vvDb []string
		err = tx.Select(&vvDb, `
			SELECT
			    pg_enum.enumlabel
			FROM
			    pg_type,
			    pg_enum
			WHERE
			    pg_enum.enumtypid = pg_type.oid AND
			    pg_type.typname = $1;`,
			typeName)
		if err != nil {
			err = tx.Rollback(err)
			panic(err)
		}

		// If type doesn't exist at all we simply create it.
		if vvDb == nil {
			if _, err = tx.Exec(s); err != nil {
				err = tx.Rollback(err)
				panic(err)
			}
			continue
		}

		// If type exists and contains the same values.
		if sameSet(vvDb, vv) {
			continue
		}

		/*
			If the type has changed we have to delete the
			old type and create a new one. We can't alter
			the type because:

				a.) altering types in transactions isn't
					possible and
				b.) even if you could alter the enum type
					you can only add values not remove them

			To delete the old type we first change the
			type of the columns using the old type. Now
			we can safely delete it. Next we create the
			the new type with the same name. Finally we
			change the column types back to the new type
			(with the same name).
		*/

		// Get all table names.
		tblNames, err := allTables(tx)
		if err != nil {
			err = tx.Rollback(err)
			panic(err)
		}

		altered := make(map[string][]string)

		for _, tblName := range tblNames {

			// Find all columns in current table of current type.
			var cols []string
			err = tx.Select(&cols, fmt.Sprintf(`
                SELECT
                    column_name
                FROM
                    INFORMATION_SCHEMA.COLUMNS
                WHERE
                    table_name = '%s' AND
                    data_type = 'USER-DEFINED' AND
                    udt_name = '%s';`,
				tblName, typeName))
			if err != nil {
				err = tx.Rollback(err)
				panic(err)
			}
			if cols == nil {
				continue
			}

			/*
				Keep track of the columns we're altering
				as we'll need to change them again to the
				new type.
			*/
			altered[tblName] = cols

			/*
				Alter all columns of type to use text. The
				column::text syntax means we cast it from
				the current enum type to text.
			*/
			for _, col := range cols {
				_, err = tx.Exec(fmt.Sprintf(`
					ALTER TABLE
						%s
					ALTER COLUMN
						%s
					TYPE
						text
					USING
						%s::text
					`, tblName, col, col))
				if err != nil {
					err = tx.Rollback(err)
					panic(err)
				}
			}
		}

		// Drop old type.
		_, err = tx.Exec(`DROP TYPE IF EXISTS ` + typeName)
		if err != nil {
			err = tx.Rollback(err)
			panic(err)
		}

		// Create new type.
		_, err = tx.Exec(s)
		if err != nil {
			err = tx.Rollback(err)
			panic(err)
		}

		/*
			Alter all columns to use the new type instead
			of text. The column::new_type syntax means we
			cast it from text to the new enum type.
		*/
		for tblName, cols := range altered {
			for _, col := range cols {
				_, err = tx.Exec(fmt.Sprintf(`
					ALTER TABLE
						%s
					ALTER COLUMN
						%s
					TYPE
						%s
					USING
						%s::%s
					`, tblName, col, typeName, col, typeName))
				if err != nil {
					err = tx.Rollback(err)
					panic(err)
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		panic(err)
	}
}

func dbTables(c *sd.Config, db sd.DB) {

	f, err := ioutil.ReadFile(c.PathDbTables)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	for _, s := range strings.SplitAfter(string(f), ";") {
		if _, err = tx.Exec(s); err != nil {
			err = tx.Rollback(err)
			panic(err)
		}
	}

	if err = tx.Commit(); err != nil {
		panic(err)
	}
}

func firstAccount(c *sd.Config, log sd.Logger, db sd.DB, as sd.Accounts) {
	rId := "INIT"
	defer log.End(rId, "", rId, "/", 0)
	acc, err := as.RetrieveByHandle(rId, c.Credentials.InitHandle, sd.AccOptRetrieve{
		Confirmed: true,
	})
	if err != nil && err != sql.ErrNoRows {
		log.Error(rId, err.Error())
		return
	}
	if acc != nil {
		return
	}
	fb, err := as.Create(
		rId,
		&sd.Registration{
			Handle:   c.Credentials.InitHandle,
			Email:    c.Credentials.InitEmail,
			Password: c.Credentials.InitPass,
		},
		sd.ConfirmAuto,
		true,
	)
	if err != nil {
		panic(err)
	}
	if len(fb) > 0 {
		for _, v := range fb {
			for _, s := range v {
				log.Error(rId, s)
			}
		}
		panic("failed to create account")
	}
}

type selector interface {
	Select(interface{}, string, ...interface{}) error
}

func allTables(x selector) (tables []string, err error) {
	err = x.Select(&tables, `
        SELECT
			tablename
        FROM
			pg_catalog.pg_tables
		WHERE
			tablename NOT LIKE 'pg_%' AND
			tablename NOT LIKE 'sql_%'
		ORDER BY
			tablename ASC;
    `)
	return tables, err
}

func superset(origSet, newSet []string) (newVals []string, ok bool) {
	for _, v := range origSet {
		if !in(newSet, v) {
			return nil, false
		}
	}
	for _, v := range newSet {
		if !in(origSet, v) {
			newVals = append(newVals, v)
		}
	}
	return newVals, true
}

func sameSet(ss1, ss2 []string) bool {
	if len(ss1) != len(ss2) {
		return false
	}
	for _, v := range ss1 {
		if !in(ss2, v) {
			return false
		}
	}
	return true
}
