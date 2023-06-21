package populate

import (
	"fmt"

	sd "github.com/jakebowkett/storydevs"
)

func Settings(category string, ed sd.Fields, p sd.Persona) (sd.Fields, error) {

	field, err := ed.Field(category)
	if err != nil {
		return nil, fmt.Errorf("populate: %w", err)
	}

	/*
		We empty .Context here otherwise it'll display
		at the top of the sub-settings' form which looks
		ugly and is redundant because .Context is used
		as a description in the results column.

		Since we're operating on a copy we can do this
		and it won't modify the original.
	*/
	field.Context = ""

	ff := field.Field
	switch category {
	case "identity":
		for i, f := range ff {
			switch f.Name {
			case "handle":
				ff[i].Text = p.Handle
			case "name":
				ff[i].Text = p.Name.String
			case "avatar":
				/*
					We check if this is the empty string here because
					the URL() method prefixes the filename with the
					directory path to the file. That is, in the template
					it will test true in conditionals even when there's
					no avatar.
				*/
				if p.Avatar.String() != "" {
					ff[i].Text = p.Avatar.URL()
				}
			case "pronouns":
				newF := f
				for _, pn := range p.Pronouns {
					newF.Text = pn
					ff[i].Instances = append(ff[i].Instances, newF)
				}
			}
		}
	case "privacy":
		for i, f := range ff {
			switch f.Name {
			case "visibility":
				ff[i].Default = p.Visibility
			}
		}
	default:
		return nil, fmt.Errorf("populate: unknown persona settings category %q", category)
	}

	return sd.Fields{*field}, nil
}
