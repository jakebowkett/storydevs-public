/*
The purpose of this program is to take the IANA zone1970.tab
file (example included in same directory as this file) and
extract its TZ database names. From these extracted names it
creates an alphabetically sorted TOML array with the database
name for timezones as well as their display names with any
restored non-ASCII characters.

There is probably a more efficient way to write this program
rather than iterating over the same data three times but it's
only run once, it's run offline, on ~400 lines of data. It's fine.

IMPORTANT: The Idx field in the generated TOML file (which is
ultimately converted to an sd.Field field) is used by compact
queries to consistently reference timezone values whose order
in the list changes due to daylight savings. Therefore any attempt
to update the timezone.toml file in /data/mode/replace MUST
preserve those indices otherwise some links to previous searches
will break. This program does not make such accommodations yet.

Lastly, the Idx field, if present, begins at 1 not 0. This is
because Go's templates treat 0 as a zero value meaning they
will test false in conditionals.
*/
package main

import (
	"bytes"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {

	// Load file.
	bb, err := ioutil.ReadFile("./zone1970.tab")
	if err != nil {
		panic(err)
	}

	// Strip comments and any zero-length lines.
	var ss []string
	for _, ln := range bytes.Split(bb, []byte("\n")) {
		if len(ln) == 0 || ln[0] == '#' {
			continue
		}
		re := regexp.MustCompile(
			`^[A-Z]+(,[A-Z]+)*\t` + // Country codes, e.g. AE,OM
				`(\+|-)\d+(\+|-)\d+\t` + // Coords, e.g. -6835+07758
				`([A-Za-z_/.-]+)` + // The TZ location we want.
				`.*`, // The rest of the line.
		)
		submatches := re.FindStringSubmatch(string(ln))
		ss = append(ss, submatches[4])
	}

	// Alphabetically sort TZ locations without slashes.
	sort.SliceStable(ss, func(i, j int) bool {
		si := strings.ToLower(strings.Replace(ss[i], "/", "", -1))
		sj := strings.ToLower(strings.Replace(ss[j], "/", "", -1))
		return si < sj
	})
	/*
		Format into TOML and make any special-case changes here.
		The following regex is good for finding characters in the
		zone1970.tab file that couldn't be included in the alpha
		and underscore names of regions+cities. (The special cases
		below need to be extended manually due to the inconsistent
		formatting of zone1970.tab's comments.)

			[^a-zA-Z0-9_ ,.":;#()\n\t/+-]
	*/
	for i, s := range ss {
		name := s
		text := s
		text = strings.Replace(text, "DumontDUrville", "Dumont d'Urville", 1)
		text = strings.Replace(text, "Tucuman", "Tucumán", 1)
		text = strings.Replace(text, "Galapagos", "Galápagos", 1)
		text = strings.Replace(text, "Aqtobe", "Aqtöbe", 1)
		text = strings.Replace(text, "Atyrau", "Atyraū", 1)
		text = strings.Replace(text, "Bahia_Banderas", "Bahía de Banderas", 1)
		text = strings.Replace(text, "Reunion", "Réunion", 1)
		text = strings.Replace(text, "_", " ", -1)
		ss[i] = "" +
			"	[[timezone.Value]]\n" +
			"		Idx  = " + strconv.Itoa(i+1) + "\n" +
			"		Name = \"" + name + "\"\n" +
			"		Text = \"" + text + "\"\n"
	}

	// Write TOML file.
	bb = []byte(strings.Join(ss, "\n"))
	if err := ioutil.WriteFile("timezone.toml", bb, 0600); err != nil {
		panic(err)
	}
}
