package query

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	sd "github.com/jakebowkett/storydevs"
)

func Parse(r *sd.Request, form sd.Fields, dep *sd.Dependencies) (map[string][]string, error) {

	query := r.Request.URL.Query()

	if len(query) == 0 {
		return nil, nil
	}

	if _, ok := query["all"]; ok {
		return nil, nil
	}

	explodedQuery, err := explodeCompactQuery(query)
	if err != nil {
		return nil, err
	}
	delete(query, "s")

	searchTerms, err := parseSearchTerms(query)
	if err != nil {
		return nil, err
	}

	mappedQuery := make(Mapped)
	if err := mapQuery(mappedQuery, explodedQuery, form, dep); err != nil {
		return nil, err
	}

	for k, v := range searchTerms {
		mappedQuery[k] = v
	}

	if dep.Config.Dev {
		if err := printMappedQuery(r, mappedQuery, form); err != nil {
			return nil, err
		}
	}

	return mappedQuery, nil
}

type Mapped = map[string][]string
type exploded struct {
	id  string
	val []string
	num []int
}

func mapQuery(mq Mapped, eq []exploded, form sd.Fields, dep *sd.Dependencies) error {

	groupCount := make(map[string]int)
	prefix := sd.NewChain()

	for _, e := range eq {

		f, err := form.FieldById(e.id)
		if err != nil {
			return err
		}
		vv := e.val
		nn := e.num

		if len(f.Field) > 1 {
			switch nn[0] {
			case 1:
				fallthrough
			case 2:
				/*
					We check here because the user controls whether
					there are matching group delimiters or not.
				*/
				if prefix.Len() < 2 {
					return errors.New("mismtaching group delimiters")
				}
				prefix.Pop()
				groupCount[prefix.Current()]++
				prefix.Pop()
				if nn[0] == 2 {
					break
				}
				fallthrough
			case 0:
				prefix.Push(f.Name)
				prefix.Push(strconv.Itoa(groupCount[prefix.Current()]))
			}
			continue
		}

		ss := []string{}
		switch f.Type {
		case "calendar", "date":
			/*
				Calendar/date search value is a unix timestamp
				with a resolution of days. We multiply it so it
				has its original resolution of seconds. Conversion
				to int64 necessary to avoid overflow for dates
				after January 19, 2038. As a special case 0 is
				treated as the present day.
			*/
			if nn[0] == 0 {
				ss = append(ss, "Present")
				break
			}
			unix := ((int64(nn[0]) * 60) * 60) * 24
			ss = append(ss, strconv.FormatInt(unix, 10))
		case "time":
			v, ok := dep.TimeMapping[vv[0]]
			if !ok {
				return fmt.Errorf("Key %q not found in time mapping", vv[0])
			}
			ss = append(ss, v)
		default:
			switch {
			/*
				If the first element of Value has a non-zero .Idx field
				we assume they are all non-zero and use them instead
				of their actual indices. This is useful for data whose
				position may change over the course of the program.

				(Perhaps all fields should work like this in the future
				to guard against changes to the forms values breaking
				searches.)
			*/
			case len(f.Value) > 0 && f.Value[0].Idx != 0:
				matches := 0
				for _, v := range f.Value {
					if !in(nn, v.Idx) {
						continue
					}
					ss = append(ss, v.Name)
					matches++
					if len(nn) == matches {
						break
					}
				}
			/*
				If the query value was modified it won't be found
				as an index value.
			*/
			case f.ValueModify != "":
				/*
					Nothing to do here. Perhaps in the future ValueModify
					should be both a client and server function, with the
					server-side version doing validation here.
				*/
				for _, n := range nn {
					ss = append(ss, strconv.Itoa(n))
				}
			default:
				// Otherwise assume it's an index in Value.
				for _, n := range nn {
					if n < 0 {
						return errors.New("query: negative index for value")
					}
					if n > len(f.Value)-1 {
						return errors.New("query: index for value out of bounds")
					}
					ss = append(ss, f.Value[n].Name)
				}
			}
		}
		prefix.Push(f.Name)
		mq[prefix.Current()] = ss
		prefix.Pop()
	}

	for name, n := range groupCount {
		mq[name] = []string{strconv.Itoa(n)}
	}

	return nil
}

func in(nn []int, n int) bool {
	for i := 0; i < len(nn); i++ {
		if nn[i] == n {
			return true
		}
	}
	return false
}

func parseSearchTerms(query url.Values) (map[string][]string, error) {
	m := make(map[string][]string)
	for k, v := range query {
		if len(v) != 1 {
			return nil, errors.New(`expected 1 element when parsing search terms`)
		}
		terms := strings.Split(v[0], ",")
		m[k] = terms
	}
	if len(m) == 0 {
		return nil, nil
	}
	return m, nil
}

/*
	explodeCompactQuery transforms a compact query into an easier
	to manipulate format. A compact query is a string assigned to
	"s" in a URL's query that alternates between letters and numbers.
	The letters ultimately correspond to form field names while the
	numbers represent values that field can hold. If the compact query
	were "A0C1.3.9.10D5" explodeCompactQuery would transform it into:

	    []exploded{
	    	{id: "A", val: []string{"0"}, num: []int{0}},
	    	{id: "C", val: []string{"1", "3", "9", "10"}, num: []int{1, 3, 9, 10}},
	    	{id: "D", val: []string{"5"}, num: []int{5}},
	    }

    Both string and integer versions are create because while
    many widget types prefer integers for their mapping process,
    the time widget uses leading zeroes for its mappings and
    those are erased when converting the string to and integer.
*/

func explodeCompactQuery(query url.Values) (m []exploded, err error) {

	ss, ok := query["s"]
	if !ok {
		return nil, nil
	}

	if len(ss) != 1 {
		return nil, errors.New(`expected 1 element for query key "s"`)
	}

	s := ss[0]

	// Assert that the format alternates between letters and numbers.
	if !cqFormat.MatchString(s) {
		return nil, errors.New(`malformed compact query`)
	}

	// Separate letters and numbers from each other.
	ss = cqSplit.FindAllString(s, -1)

	for i := range ss {

		// Even indices are letters.
		if i%2 == 0 {
			continue
		}

		// Odd is numbers. When there are multiple
		// numbers they are separated by a period.
		vv := strings.Split(ss[i], ".")
		nn := []int{}
		for _, v := range vv {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			nn = append(nn, n)
		}
		m = append(m, exploded{
			id:  ss[i-1],
			val: vv,
			num: nn,
		})
	}

	return m, nil
}

var cqFormat = regexp.MustCompile(`^([A-Za-z]+-?\d+(\.-?\d+)*)+$`)
var cqSplit = regexp.MustCompile(`[A-Za-z]+|-?\d+(\.-?\d+)*`)

func printMappedQuery(r *sd.Request, mq Mapped, form sd.Fields) error {

	v, ok := r.Vars["mode"]
	inForums := ok && v == "forums"
	if inForums {
		fmt.Printf("%24s: %s\n", "category", mq["category"])
		println("")
		return nil
	}

	kk := []string{}
	for k := range mq {
		kk = append(kk, k)
	}
	sort.Strings(kk)
	re := regexp.MustCompile(`\.\d+`)
	for _, origKey := range kk {
		var vv []string
		for _, origVal := range mq[origKey] {
			vv = append(vv, origVal)
		}
		k := re.ReplaceAllString(origKey, "")
		f, err := form.Field(k)
		if err != nil {
			return err
		}
		if len(f.Field) > 0 {
			goto skip
		}
		if f.Type == "date" || f.Type == "time" {
			goto skip
		}
		for i := range vv {
			v, err := form.Value(k, vv[i])
			if err != nil {
				return err
			}
			vv[i] = v.Text
		}
	skip:
		fmt.Printf("%24s: %s\n", origKey, strings.Join(vv, ", "))
	}
	println("")
	return nil
}
