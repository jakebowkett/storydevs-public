package populate

import (
	"fmt"
	"html/template"
	"time"

	// "github.com/davecgh/go-spew/spew"
	sd "github.com/jakebowkett/storydevs"
)

func Account(c *sd.Config, as sd.Fields, acc sd.Account) error {

	// Set created.
	f, err := as.Field("account.created")
	if err != nil {
		return fmt.Errorf("populate: %w", err)
	}
	layout := "Mon Jan 2, 2006 3:04"
	t := time.Unix(acc.Created, 0)
	if t.Hour() < 12 {
		layout += " AM"
	} else {
		layout += " PM"
	}
	f.Default = t.Format(layout)

	// Set email.
	f, err = as.Field("account.email")
	if err != nil {
		return fmt.Errorf("populate: %w", err)
	}
	f.Default = acc.Email

	// Set personas.
	f, err = as.Field("account.personas")
	if err != nil {
		return fmt.Errorf("populate: %w", err)
	}
	f.Add = c.MaxPersonas
	for i, p := range acc.Personas {
		if err := as.AddFieldInstance("account.personas"); err != nil {
			return err
		}
		f, err = as.Field(fmt.Sprintf("account.personas.%d", i))
		if err != nil {
			return err
		}
		/*
			We check if this is the empty string here because
			the URLThumb() method prefixes the filename with the
			directory path to the file. That is, in the template
			it will test true in conditionals even when there's
			no avatar.
		*/
		if p.Avatar.String() != "" {
			f.Icon = template.HTML(p.Avatar.URLThumb())
		}
		f.Value = append(f.Value, sd.Value{
			Name: p.Slug,
			Text: p.Handle,
			Desc: p.Name.String,
			True: p.Active,
		})
	}

	return nil
}
