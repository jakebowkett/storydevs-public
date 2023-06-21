package form

import (
	"fmt"

	sd "github.com/jakebowkett/storydevs"
)

func talent(p *sd.Profile, ff sd.Fields) error {

	// Check a given advertised skill and its media kind match up.
	skills, err := ff.Field("project.role.skill")
	if err != nil {
		return err
	}

	exInstances := make(map[string]int)
	seenSkill := make(map[string]bool)

	for _, ad := range p.Advertised {

		/*
			This check ensures there isn't multiple groupings of
			the same advertised skills which will result in the
			portfolio tab having duplicates (i.e., two Character
			Art tabs).
		*/
		if seenSkill[ad.Skill] {
			return fmt.Errorf("skill %q has duplicate(s)", ad.Skill)
		}
		seenSkill[ad.Skill] = true

		v, err := skills.ValueByName(ad.Skill)
		if err != nil {
			return err
		}
		for _, ex := range ad.Example {
			exInstances[ad.Skill]++
			if v.Data != ex.Kind {
				args := []interface{}{ad.Skill, v.Data, ex.Kind}
				return fmt.Errorf("skill %q only allows kind %q, got %q", args...)
			}
		}
	}

	// Check that the number of examples per skill do not exceed max.
	examples, err := ff.Field("advertised.example")
	if err != nil {
		return err
	}
	for skill := range exInstances {
		if n := exInstances[skill]; n != examples.Add {
			args := []interface{}{skill, examples.Add, n}
			return fmt.Errorf("example %q expected %d instances, got %d", args...)
		}
	}

	return nil
}
