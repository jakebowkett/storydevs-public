package setup

import (
	"bytes"
	"fmt"
	"html/template"
	txtTemplate "html/template"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jakebowkett/go-num/num"
	sd "github.com/jakebowkett/storydevs"
)

func mustViewData(c *sd.Config, cache sd.Cache, h sd.Hyphenator) *sd.ViewData {

	sharedData := mustParseSharedData(c.DirShared, cache)

	replacer := sd.Replace{
		Dir:  c.DirReplace,
		Load: unmarshalTOML,
	}

	mp := modeParser{
		config:    c,
		cache:     cache,
		skipId:    []string{"text", "textarea", "tagger", "keyworder"},
		shared:    sharedData,
		replace:   replacer,
		hyphenate: h,
	}

	modeData := mp.mustParseModeData(c.DirMode)
	pageData := mustParsePageData(c.DirPage)
	modalData := mustParseModalData(c.DirModal, cache, c, h, replacer, sharedData)
	errorData := mustParseErrorData(c.DirError)

	/*
		Save initialised copies to load later when
		populating search and editor UI for modes.
	*/
	for mode, data := range modeData {

		search := fmt.Sprintf("ui/mode/%s_search_init", mode)
		editor := fmt.Sprintf("ui/mode/%s_editor_init", mode)

		s := struct{ Search sd.Fields }{data.Search}
		e := struct{ Editor sd.Fields }{data.Editor}

		buf := new(bytes.Buffer)
		if err := toml.NewEncoder(buf).Encode(s); err != nil {
			panic(err)
		}
		cache.AddString(search, buf.String())

		buf.Reset()

		if err := toml.NewEncoder(buf).Encode(e); err != nil {
			panic(err)
		}
		cache.AddString(editor, buf.String())
	}

	vd := &sd.ViewData{}
	vd.Page = pageData
	vd.Mode = modeData
	vd.Modal = modalData
	vd.Errors = errorData
	vd.Hyphenator = h.Hyphenate

	return vd
}

func mustParseErrorData(dirPath string) map[string]string {
	m := mustParseData(dirPath, func() interface{} { return &map[string]string{} })
	return *m["errors"].(*map[string]string)
}

func mustParsePageData(dirPath string) sd.Page {

	dataMap := mustParseData(dirPath, func() interface{} { return &sd.PageData{} })
	m := make(sd.Page, len(dataMap))

	for name, data := range dataMap {
		// The instance function for mustParseData has to return a
		// pointer to sd.ModeData, sd.PageData, and sd.ModalData
		// otherwise the toml data will not be unmarshalled into those
		// structs; instead it will be unmarshalled into a map of
		// strings to empty interfaces. We dereference the pointer
		// here so that the request handlers will operate on copies
		// of the data rather than modifying the originals.
		m[name] = *data.(*sd.PageData)
	}

	return m
}

func replaceOverride(f *sd.Field, field []sd.Field, i int) {
	if field[i].Name != "" {
		f.Name = field[i].Name
	}
	if field[i].Desc != "" {
		f.Desc = field[i].Desc
	}
	if field[i].Placeholder != "" {
		f.Placeholder = field[i].Placeholder
	}
	if field[i].Default != "" {
		f.Default = field[i].Default
	}
	if field[i].Context != "" {
		f.Context = field[i].Context
	}
	if field[i].Help != "" {
		f.Help = field[i].Help
	}
	if field[i].Type != "" {
		f.Type = field[i].Type
	}
	if field[i].ValueSet != "" {
		f.ValueSet = field[i].ValueSet
	}
	if field[i].ValueModify != "" {
		f.ValueModify = field[i].ValueModify
	}
	if field[i].Hidden {
		f.Hidden = true
	}
	if field[i].Paired {
		f.Paired = true
	}
	if field[i].Optional {
		f.Optional = true
	}
	if field[i].ServerOnly {
		f.ServerOnly = true
	}
}

func mustParseModalData(
	path string,
	cache sd.Cache,
	c *sd.Config,
	h sd.Hyphenator,
	replace sd.Replace,
	shared sd.Shared,
) sd.Modal {

	dataMap := mustParseData(path, func() interface{} { return &sd.ModalData{} })
	m := make(sd.Modal, len(dataMap))

	for name, data := range dataMap {

		// The instance function for mustParseData has to return a
		// pointer to sd.ModeData, sd.PageData, and sd.ModalData
		// otherwise the toml data will not be unmarshalled into those
		// structs; instead it will be unmarshalled into a map of
		// strings to empty interfaces. We dereference the pointer
		// here so that the request handlers will operate on copies
		// of the data rather than modifying the originals.
		d := *data.(*sd.ModalData)

		for i := range d.Field {

			if r := d.Field[i].Replace; r != "" {
				f, err := replace.InstanceOf(r)
				if err != nil {
					panic(err)
				}
				replaceOverride(&f, []sd.Field(d.Field), i)
				d.Field[i] = f
			}

			if sh := d.Field[i].Shared; sh != "" {
				val, ok := shared[sh]
				if !ok {
					panic(fmt.Sprintf("unable to find %q in shared", sh))
				}
				vv := make([]sd.Value, len(val))
				for i, v := range val {
					vv[i] = v
				}
				d.Field[i].Value = vv
			}

			if help, ok := c.InputHelp[d.Field[i].Type]; ok {
				d.Field[i].Help = h.Hyphenate(help)
			}
			d.Field[i].Context = h.Hyphenate(d.Field[i].Context)
			setFieldNote(d.Field, i)
		}

		for i := range d.Button {
			d.Button[i].Icon = icon(d.Button[i].Icon, cache)
		}

		m[name] = d
	}

	return m
}

type modeParser struct {
	mode           string
	modeGroup      string
	resourcePlural string // plural resource name, e.g. "Entries", "Profiles", etc
	config         *sd.Config
	cache          sd.Cache
	skipId         []string
	fieldNum       int
	shared         sd.Shared
	replace        sd.Replace
	path           []string
	hyphenate      sd.Hyphenator
}

func (mp *modeParser) popPath() {
	mp.path = mp.path[0 : len(mp.path)-1]
}
func (mp *modeParser) pushPath(seg string) {
	mp.path = append(mp.path, seg)
}

func (mp *modeParser) mustParseModeData(dirPath string) sd.Mode {

	dataMap := mustParseData(dirPath, func() interface{} { return &sd.ModeData{} })
	m := make(sd.Mode, len(dataMap))

	for name, data := range dataMap {

		// The instance function for mustParseData has to return a
		// pointer to sd.ModeData, sd.PageData, and sd.ModalData
		// otherwise the toml data will not be unmarshalled into those
		// structs; instead it will be unmarshalled into a map of
		// strings to empty interfaces. We dereference the pointer
		// here so that the request handlers will operate on copies
		// of the data rather than modifying the originals.
		d := *data.(*sd.ModeData)
		forms := []sd.Fields{d.Search, d.Editor}
		mp.mode = d.Name
		mp.resourcePlural = strings.ToLower(d.ResourcePlural)

		for i := range forms {

			if i == 0 {
				mp.modeGroup = "search"
			} else {
				mp.modeGroup = "editor"
			}

			form := forms[i]
			mp.fieldNum = 0

			seen := map[string]bool{}

			for i := range form {

				mp.pushPath(form[i].Name)
				mp.checkName("group", form[i].Name, form[i].Desc, seen)

				form[i].Context = mp.hyphenate.Hyphenate(form[i].Context)
				form[i].Icon = icon(form[i].Icon, mp.cache)
				form[i].AddIcon = icon(form[i].AddIcon, mp.cache)
				setFieldNote(form, i)

				mp.recurseIntoSubgroups(form[i].Field)
				mp.popPath()
			}

			/*
				We do this after the above stuff to ensure
				replaced fields are added before our checks.
			*/
			uniqueTopLevelFieldNames(form)
		}

		m[name] = d
	}

	return m
}

func uniqueTopLevelFieldNames(mg sd.Fields) error {
	seen := make(map[string]bool)
	for _, g := range mg {
		if g.Add > 0 {
			continue
		}
		for _, f := range g.Field {
			if seen[f.Name] {
				return fmt.Errorf("Top level field %s doesn't have unique name.", f.Name)
			}
			seen[f.Name] = true
		}
	}
	return nil
}

func (mp *modeParser) recurseIntoSubgroups(field []sd.Field) {
	// We call checkName after parsing below to allow for field substitutions.
	seen := map[string]bool{}
	for i := range field {
		mp.pushPath(field[i].Name)
		mp.parseField(field, i)
		mp.checkName("field", field[i].Name, field[i].Desc, seen)
		mp.recurseIntoSubgroups(field[i].Field)
		mp.popPath()
	}
}

func (mp *modeParser) checkName(kind, name, desc string, seen map[string]bool) {
	args := []interface{}{mp.mode, mp.modeGroup, kind, desc}
	if name == "" {
		log.Panicf("%s %s %s %q has no name", args...)
	}
	if r := regexp.MustCompile(`^\w+$`); !r.MatchString(name) {
		log.Panicf("%s %s %s %q contains non-alphanumeric character", args...)
	}
	args[2] = name
	args = args[0:3]
	if _, ok := seen[name]; ok {
		log.Panicf("%s %s field name %q already in use in this form", args...)
	}
	seen[name] = true
}

func (mp *modeParser) parseField(field []sd.Field, i int) {

	if replace := field[i].Replace; replace != "" {
		f, err := mp.replace.InstanceOf(replace)
		if err != nil {
			panic(err)
		}
		replaceOverride(&f, field, i)
		field[i] = f
	}

	field[i].Icon = icon(field[i].Icon, mp.cache)

	// We check .Field's length to ensure we skip subgroups
	if strings.Contains(field[i].Name, " ") && len(field[i].Field) == 0 {
		log.Panicf(
			`field name %q in the %s %s contains a space which is disallowed `+
				`due to field names being used as HTML classes`,
			field[i].Name,
			mp.mode,
			mp.modeGroup,
		)
	}

	t, err := txtTemplate.New("context").Parse(field[i].Context)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = t.Execute(&buf, mp.resourcePlural)
	if err != nil {
		panic(err)
	}
	field[i].Context = mp.hyphenate.Hyphenate(buf.String())

	if help, ok := mp.config.InputHelp[field[i].Type]; ok {
		field[i].Help = help
		// setImageHelp(field, i)
	}

	setFieldNote(field, i)

	mp.parseEvents(field, i)

	if shared := field[i].Shared; shared != "" {

		val, ok := mp.shared[shared]
		if !ok {
			panic(fmt.Sprintf("unable to find %q in Shared", shared))
		}

		vv := make([]sd.Value, len(val))
		for i, v := range val {
			vv[i] = v
		}

		field[i].Value = vv
	}

	// We use "j" here so we can access the outer "i".
	val := field[i].Value
	for j := range val {

		if val[j].Name == "" {
			log.Panicf("%s in %s %s has values with no names", field[i].Name, mp.mode, mp.modeGroup)
		}

		// Shared data already has rendered icons.
		if field[i].Shared == "" {
			val[j].Icon = icon(val[j].Icon, mp.cache)

			nested := val[j].Value
			for k := range nested {
				nested[k].Icon = icon(nested[k].Icon, mp.cache)
			}
		}

		if val[j].Icon != "" {
			field[i].HasIcons = true
		}

		for _, vd := range val[j].ValueData {
			if strings.Join(mp.path, ".") == vd.Field {
				val[j].Data = vd.Data
			}
		}
	}

	// Since groups are untyped they are also assigned IDs.
	if !in(mp.skipId, field[i].Type) {
		id, err := num.Alpha(mp.fieldNum)
		if err != nil {
			panic(err)
		}
		field[i].Id = id
		mp.fieldNum++
	}
}

func (mp *modeParser) parseEvents(field []sd.Field, i int) {

	ee := field[i].Events
	sep := ";"

	for i := range ee {

		for _, arg := range ee[i].Args {

			if strings.Contains(arg, sep) {
				panic(fmt.Sprintf(
					"%s in %s %s: Event arguments cannot contain %q as it is used to separate arguments.",
					field[i].Name,
					mp.mode,
					mp.modeGroup,
					sep,
				))
			}
		}

		args := strings.Join(ee[i].Args, sep)
		if args != "" {
			ee[i].Handler += "[" + args + "]"
		}
	}
}

// func setImageHelp(field []sd.Field, i int) {
// 	whitelist := []string{"image", "thumb"}
// 	if !in(whitelist, field[i].Type) {
// 		return
// 	}
// 	if field[i].Max == 0 {
// 		return
// 	}
// 	max := num.Bytes(field[i].Max * 1024)
// 	field[i].Help = fmt.Sprintf(field[i].Help, max)
// }

func setFieldNote(field []sd.Field, i int) {

	var ss []string
	f := field[i]

	if f.AddMin > f.Add {
		panic(".AddMin is greater than .Add")
	}
	if f.Max != 0 && f.Min > f.Max {
		panic(".Min is greater than .Max")
	}

	if f.Optional {
		ss = append(ss, "Optional")
	}

	if f.Add > 0 {
		switch {
		case f.AddMin == 0:
			ss = append(ss, fmt.Sprintf("Add up to %d", f.Add))
		case f.AddMin == f.Add:
			ss = append(ss, fmt.Sprintf("Add %d", f.Add))
		default:
			ss = append(ss, fmt.Sprintf("Add between %d and %d", f.AddMin, f.Add))
		}
	}

	skip := []string{
		"text",
		"textarea",
		"image",
		"thumb",
		"newpassword",
		"editor",
		"tagger",
		"keyworder",
	}
	if !in(skip, f.Type) {
		if f.Min > 0 {
			ss = append(ss, fmt.Sprintf("Minimum %d", f.Min))
		}
		if f.Max > 0 {
			ss = append(ss, fmt.Sprintf("Maximum %d", f.Max))
		}
	}
	if f.Type == "newpassword" && f.Min > 0 {
		ss = append(ss, fmt.Sprintf("Minimum %d", f.Min))
	}

	if (f.Type == "image" || f.Type == "thumb") && f.Max > 0 {
		ss = append(ss, "JPEG or PNG")
		ss = append(ss, fmt.Sprintf("Max Size %s", num.Bytes(f.Max*1024)))
	}

	/*
	   We set .Note by indexing the original field
	   slice instead of our convenience copy.
	*/
	field[i].Note = strings.Join(ss, ", ")
}

func addMsg(min, max int) (msg string) {
	if min > max {
		panic(".AddMin is greater than .Add")
	}
	if max == 0 {
		return msg
	}
	if min == 0 {
		return fmt.Sprintf("Add up to %d", max)
	}
	if min == max {
		return fmt.Sprintf("Add %d", max)
	}
	return fmt.Sprintf("Add between %d and %d", min, max)
}

func mustParseSharedData(dirPath string, cache sd.Cache) sd.Shared {

	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		panic(err)
	}

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	var data []byte

	for _, info := range dir {

		if !info.Mode().IsRegular() {
			continue
		}

		f, err := ioutil.ReadFile(filepath.Join(dirPath, info.Name()))
		if err != nil {
			panic(err)
		}

		data = append(data, f...)
	}

	m := make(map[string][]sd.Value)
	if err := toml.Unmarshal(data, &m); err != nil {
		panic(err)
	}

	for name := range m {
		for i := range m[name] {
			m[name][i].Icon = icon(m[name][i].Icon, cache)
		}
	}

	return m
}

func unmarshalTOML(dirPath string, data interface{}) error {

	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return err
	}

	var concatFiles []byte

	for _, info := range dir {

		if !info.Mode().IsRegular() {
			continue
		}

		f, err := ioutil.ReadFile(filepath.Join(dirPath, info.Name()))
		if err != nil {
			return err
		}

		concatFiles = append(concatFiles, f...)
	}

	return toml.Unmarshal(concatFiles, data)
}

func mustParseData(
	dirPath string,
	instance func() interface{},
) map[string]interface{} {

	mode := strings.HasSuffix(dirPath, "mode")

	dirPath, err := filepath.Abs(dirPath)
	if err != nil {
		panic(err)
	}

	dir, err := ioutil.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	dataMap := make(map[string]interface{})

	for _, info := range dir {

		if !info.Mode().IsRegular() {
			continue
		}

		f, err := ioutil.ReadFile(filepath.Join(dirPath, info.Name()))
		if err != nil {
			panic(err)
		}

		name := strings.TrimSuffix(info.Name(), ".toml")
		if mode {
			name = strings.Split(name, "_")[0]
		}

		if _, ok := dataMap[name]; !ok {
			dataMap[name] = []byte{}
		}

		dataMap[name] = append(dataMap[name].([]byte), f...)
	}

	for name, concatFiles := range dataMap {

		data := instance()

		err = toml.Unmarshal(concatFiles.([]byte), data)
		if err != nil {
			panic(err)
		}

		dataMap[name] = data
	}

	return dataMap
}

func icon(alias template.HTML, cache sd.Cache) template.HTML {
	i := cache.Load("svg/" + string(alias) + ".svg")
	if i == nil {
		return ""
	}
	return i.HTML()
}

func in(ss []string, s string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}
