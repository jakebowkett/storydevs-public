package mode

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
	"github.com/jakebowkett/storydevs/handler/httperr"
	"github.com/jakebowkett/storydevs/handler/mode/populate"
	"github.com/jakebowkett/storydevs/handler/mode/query"
)

type columns struct {
	Search bool `json:"search"`
	Browse bool `json:"browse"`
	Detail bool `json:"detail"`
	Editor bool `json:"editor"`
}

func Full(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger
	view := dep.Templates
	cache := dep.Cache
	vd := dep.ViewData

	return func(w http.ResponseWriter, r *sd.Request) {

		mode := r.Vars["mode"]
		slug := r.Vars["resource"]
		submode := r.Vars["submode"]

		need := neededColumns(r)

		cols, resource, err := renderPartial(r, need, dep)
		if err != nil {
			log.Error(r.Id, err.Error())
			r.Status = 404
			httperr.Error(dep)(w, r)
			return
		}

		base, err := handler.Base(c, cache, r)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		base.View = mode
		base.SubView = submode
		base.ViewType = "mode"
		base.ViewMeta = vd
		base.Layout = layout(mode, slug, need)

		if slug != "new" {
			base.ResourceSlug = slug
		}
		if need.Editor || submode == "settings" {
			base.Editing = base.ResourceSlug
		}

		base.Title = vd.Mode[mode].Title
		if submode != "" {
			base.Title += " | " + vd.Mode[submode].ResourcePlural
		}

		if resource != nil {
			base.ResourceOwner = resource.IsOwner(base.Account)
			base.Title = resource.GetName()
			setMeta(base, resource)
		}

		resourceKind := vd.Mode[mode].ResourceColumn
		base.Search.Content = cols["search"]
		base.Browse.Content = renderEmpty(c, cols, "browse", resourceKind)
		base.Detail.Content = renderEmpty(c, cols, "detail", resourceKind)
		base.Editor.Content = renderEmpty(c, cols, "editor", resourceKind)

		v, err := view.Render("base.html", base)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		// if len(cols) == 1 {
		// 	handler.CacheControl(c, w, c.CacheHTML.Seconds(), sd.CachePrivate)
		// }
		handler.Gzip(w, r, v, 200, log)
	}
}

func setMeta(base *sd.Base, resource sd.Resource) {

	base.Title = resource.GetName()
	base.MetaTwitter = ""

	switch tp := resource.(type) {

	case *sd.Profile:

		base.MetaDesc = tp.Summary.String

		// Attempt to find a user supplied image, prefering landscape.
		var firstImage string
		for _, ad := range tp.Advertised {
			for _, ex := range ad.Example {
				if ex.Kind != "image" {
					continue
				}
				if ex.Aspect >= 1.5 {
					base.MetaCard = ex.File.Name.URLFull()
					base.MetaAlt = ex.AltText
					break
				}
				if firstImage == "" {
					base.MetaCard = ex.File.Name.URLFull()
					base.MetaAlt = ex.AltText
				}
			}
		}
	}
}

func renderEmpty(
	c *sd.Config,
	cols map[string]template.HTML,
	col string,
	resource string,
) template.HTML {

	if _, ok := cols[col]; ok {
		return cols[col]
	}

	a := "a"
	if resource != "" && strings.ContainsAny(resource[0:1], "AEIOU") {
		a = "an"
	}

	resource = strings.ToLower(resource)

	newCol := strings.Replace(c.Empty[col], "{{a}}", a, -1)
	newCol = strings.Replace(newCol, "{{resource}}", resource, -1)
	newCol = strings.Replace(newCol, "\n", "<br>", -1)
	newCol = `<div class="empty">` + newCol + `</div>`

	return template.HTML(newCol)
}

func Partial(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger

	return func(w http.ResponseWriter, r *sd.Request) {

		need := neededColumn(r)

		cols, resource, err := renderPartial(r, need, dep)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		var col template.HTML
		for _, c := range cols {
			col = c
			break
		}

		response := struct {
			Owner bool          `json:"owner,omitempty"`
			Col   template.HTML `json:"col"`
		}{
			Col: col,
		}

		if user, ok := r.User.(sd.Account); ok && resource != nil {
			response.Owner = resource.IsOwner(user)
		}

		p, err := json.Marshal(response)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		w.Header().Add("Content-Type", "application/json")
		// if need.Search {
		// 	handler.CacheControl(c, w, c.CacheHTML.Seconds(), sd.CachePublic)
		// }
		handler.Gzip(w, r, p, 200, log)
	}
}

/*
We parse search/editor forms anew each request because we need
to allow them to be updated. Writing to an sd.Fields when it's
potentially being read in multiple templates at once won't work.
So we just have distinct copies.
*/
func renderPartial(
	r *sd.Request,
	need columns,
	dep *sd.Dependencies,
) (
	map[string]template.HTML,
	sd.Resource,
	error,
) {

	c := dep.Config
	cache := dep.Cache
	view := dep.Templates
	vd := dep.ViewData
	rs := dep.Resources

	var resource sd.Resource
	cols := make(map[string]template.HTML, 4)

	mode := r.Vars["mode"]
	slug := r.Vars["resource"]
	data := vd.Mode[mode]
	inAccount := mode == "account"
	inAdmin := mode == "admin"
	inForums := mode == "forums"

	inSubMode := false
	if sm, ok := r.Vars["submode"]; ok {
		mode = sm
		inSubMode = true
	}
	inSettings := mode == "settings"

	if inSettings && slug != "" {
		need.Detail = false
		need.Editor = false
	}

	data.Name = mode
	data.InAdmin = inAdmin
	data.InAccount = inAccount
	data.Editor = vd.Mode[mode].Editor
	data.ResourceName = vd.Mode[mode].ResourceName
	data.ResourcePlural = vd.Mode[mode].ResourcePlural

	path := strings.TrimSuffix(r.Request.URL.Path, "/partial")

	account, ok := r.User.(sd.Account)
	var activePersona sd.Persona
	admin := false
	if ok {
		data.Account = account
		activePersona = account.ActivePersona()
		admin = activePersona.Admin.Bool
		data.IsAdmin = admin
	}

	if slug == "new" {
		data.EditorKind = "new"
	}
	if strings.HasSuffix(path, "/edit") || inSettings {
		data.EditorKind = "edit"
	}
	if strings.HasSuffix(path, "/reply") {
		data.EditorKind = "reply"
	}

	mappedQuery, err := query.Parse(r, vd.Mode[mode].Search, dep)
	if err != nil {
		return nil, nil, err
	}
	if inSubMode {
		mappedQuery = make(map[string][]string)
		mappedQuery["menu"] = []string{mode}
	}

	if need.Search {

		clearDefaults := mappedQuery != nil

		cols["search"] = ""

		/*
			We make a copy of the search so we don't
			have to worry about it persisting between
			unrelated requests.
		*/
		switch {
		case inAdmin:
			search, err := populate.Copy("admin", "search", dep, clearDefaults)
			if err != nil {
				return nil, nil, err
			}
			if err := queryToSearch(search, mappedQuery); err != nil {
				return nil, nil, err
			}
			data.Search = search
		case inAccount:
			search, err := populate.Copy("account", "search", dep, clearDefaults)
			if err != nil {
				return nil, nil, err
			}
			if err := populate.Account(c, search, account); err != nil {
				return nil, nil, err
			}
			if err := queryToSearch(search, mappedQuery); err != nil {
				return nil, nil, err
			}
			data.Search = search
		default:
			search, err := populate.Copy(mode, "search", dep, clearDefaults)
			if err != nil {
				return nil, nil, err
			}
			if len(mappedQuery) != 0 || inForums {
				if inForums {
					rm := rs["forums"].(sd.ResourceMeta)
					err := rm.Meta(r.Id, "forums", search)
					if err != nil {
						return nil, nil, err
					}
				}
				if err := queryToSearch(search, mappedQuery); err != nil {
					return nil, nil, err
				}
			}
			data.Search = search
		}
	}

	if need.Browse {

		var results []sd.Resource

		switch {
		case inAdmin:
			break
		case inAccount:
			persId := activePersona.Id
			mappedQuery = make(map[string][]string)
			mappedQuery["persona"] = []string{strconv.FormatInt(persId, 10)}
			mappedQuery["deleted"] = []string{"false"}
		default:
			if mappedQuery == nil {
				mappedQuery = make(map[string][]string)
			}
			mappedQuery["thread"] = []string{"true"}
			if !in([]string{"forums", "library"}, mode) {
				mappedQuery["visibility"] = []string{"public"}
				mappedQuery["persona_visibility"] = []string{"public"}
				mappedQuery["deleted"] = []string{"false"}
			}
		}

		results, err = rs[mode].Filter(r.Id, admin, mappedQuery)
		if err != nil {
			return nil, nil, err
		}
		data.Results = results
		cols["browse"] = ""
	}

	if need.Detail {

		var err error
		id := idFromSlug(slug)

		resource, err = rs[mode].Retrieve(r.Id, id, sd.ResOpts{
			GetPrivate: true,
		})
		if err != nil {
			return nil, nil, err
		}

		if !admin && !mayAccessResource(resource, account) {
			return nil, nil, errors.New("resource access denied")
		}

		data.Resource = resource
		cols["detail"] = ""
	}

	if need.Editor {

		/*
			We make a copy of the editor so we don't
			have to worry about it persisting between
			unrelated requests.
		*/
		ed, err := populate.Copy(mode, "editor", dep, false)
		if err != nil {
			return nil, nil, err
		}
		data.Editor = ed

		switch data.EditorKind {

		case "edit":
			slug := idFromSlug(slug)
			if resource == nil {
				resource, err = rs[mode].Retrieve(r.Id, slug, sd.ResOpts{
					GetPrivate: true,
				})
				if err != nil {
					return nil, nil, err
				}
			}
			if !admin && !mayEditResource(resource, account) {
				return nil, nil, errors.New("permission to edit resource denied")
			}
			err := populate.Editor(ed, mode, cache, resource)
			if err != nil {
				return nil, nil, err
			}
			if resource.IsReply() {
				ed, err = filterEditorForReply(ed, slug)
				if err != nil {
					return nil, nil, err
				}
			}
			data.Resource = resource
			data.Editor = ed

		// Set field "threadslug" to resource's slug.
		case "reply":
			slug := idFromSlug(slug)
			if resource == nil {
				resource, err = rs[mode].Retrieve(r.Id, slug, sd.ResOpts{
					GetPrivate: true,
				})
				if err != nil {
					return nil, nil, err
				}
			}
			if !admin && resource.IsLocked() {
				return nil, nil, errors.New("permission to reply denied: resource is locked")
			}
			ed, err = filterEditorForReply(ed, slug)
			if err != nil {
				return nil, nil, err
			}
			data.Resource = resource
			data.Editor = ed
		}

		cols["editor"] = ""
	}

	if inSettings {
		if slug != "" {
			ed, err := populate.Settings(slug, data.Editor, activePersona)
			if err != nil {
				return nil, nil, err
			}
			data.Editor = ed
			cols["detail"] = ""

		} else {
			data.Editor = nil
		}
	}

	setHyphenator(&data, vd)

	for col := range cols {
		c, err := view.Render(col+".html", data)
		if err != nil {
			return nil, nil, err
		}
		cols[col] = template.HTML(c)
	}

	return cols, resource, nil
}

func setHyphenator(data *sd.ModeData, vd *sd.ViewData) {
	if r := data.Resource; r != nil {
		r.SetHyphenator(vd.Hyphenator)
		if t, ok := r.(sd.Threader); ok {
			for _, reply := range t.Items() {
				reply.SetHyphenator(vd.Hyphenator)
			}
		}
	}
	if rr := data.Results; rr != nil {
		for _, r := range rr {
			r.SetHyphenator(vd.Hyphenator)
			if t, ok := r.(sd.Threader); ok {
				for _, reply := range t.Items() {
					reply.SetHyphenator(vd.Hyphenator)
				}
			}
		}
	}
}

func filterEditorForReply(ed sd.Fields, slug string) (sd.Fields, error) {
	f, err := ed.Field("body.threadslug")
	if err != nil {
		return nil, err
	}
	f.Text = slug
	for i := range ed {
		if ed[i].Name == "body" {
			continue
		}
		ed[i].ServerOnly = true
	}
	return ed, nil
}

func mayEditResource(r sd.Resource, a sd.Account) bool {
	return r.IsOwner(a) && !r.IsLocked()
}

func mayAccessResource(r sd.Resource, a sd.Account) bool {
	if r.GetVisibility() != sd.VisibilityPrivate {
		return true
	}
	return r.IsOwner(a)
}

func layout(mode, slug string, need columns) string {
	inAccount := mode == "account"
	inAdmin := mode == "admin"
	if need.Editor {
		if slug == "new" {
			return "search-editor"
		} else {
			return "editor"
		}
	}
	if need.Detail {
		if inAccount || inAdmin {
			return "detail"
		} else {
			return "search-detail"
		}
	}
	if need.Browse {
		return "browse"
	}
	return "search"
}

func neededColumns(r *sd.Request) columns {
	need := columns{Search: true}
	q := r.Request.URL.Query()
	if len(q) > 0 {
		need.Browse = true
	}
	if _, ok := r.Vars["submode"]; ok {
		need.Browse = true
	}
	resource, ok := r.Vars["resource"]
	if ok && resource != "new" {
		need.Detail = true
	}
	if resource == "new" {
		need.Editor = true
	}
	path := r.Request.URL.Path
	if strings.HasSuffix(path, "/edit") {
		need.Editor = true
	}
	if strings.HasSuffix(path, "/reply") {
		need.Editor = true
	}
	return need
}

func neededColumn(r *sd.Request) columns {

	need := columns{}
	path := strings.TrimSuffix(r.Request.URL.Path, "/partial")

	if strings.HasSuffix(path, "/edit") {
		need.Editor = true
		return need
	}
	if strings.HasSuffix(path, "/reply") {
		need.Editor = true
		return need
	}
	resource, ok := r.Vars["resource"]
	if resource == "new" {
		need.Editor = true
		return need
	}
	if ok {
		need.Detail = true
		return need
	}

	// For account modes.
	if _, ok := r.Vars["submode"]; ok {
		need.Browse = true
		return need
	}

	q := r.Request.URL.Query()
	if len(q) > 0 {
		need.Browse = true
		return need
	}
	need.Search = true
	return need
}

func idFromSlug(slug string) string {
	parts := strings.Split(slug, "-")
	return parts[len(parts)-1]
}

func in(ss []string, s string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}

/*
queryToSearch modifies search so that it reflects
the options chosen in mappedQuery. It is important
that the caller use a copy of a mode's search rather
than the original otherwise the modifications made
will persist between requests.
*/
func queryToSearch(search sd.Fields, mq query.Mapped) error {
	var keys []string
	for k := range mq {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, path := range keys {
		vv := mq[path]
		f, err := search.FindOrAddInstance(path)
		if err != nil {
			return err
		}
		if err := querySetField(mq, f, vv); err != nil {
			return err
		}
	}
	return nil
}

func querySetField(mq query.Mapped, f *sd.Field, val []string) error {

	switch f.Type {
	case "range":
		f.Text = strings.Join(val, "-")
		return nil
	case "tagger", "keyworder":
		f.Value = populateValue(val)
		return nil
	case "text", "textarea", "time":
		if len(val) == 0 {
			return nil
		}
		if len(val) > 1 {
			return fmt.Errorf("expected single value, got %d", len(val))
		}
		f.Text = val[0]
		return nil
	case "dropdown":
		if len(val) == 0 {
			return nil
		}
		if len(val) > 1 {
			return fmt.Errorf("expected single value, got %d", len(val))
		}
		for _, v := range f.Value {
			if v.Name == val[0] {
				var ss []string
				for i := range v.Value {
					ss = append(ss, v.Value[i].Text)
				}
				ss = append(ss, v.Text)
				f.Text = strings.Join(ss, " ")
				return nil
			}
		}
	case "calendar", "date":
		if len(val) == 0 {
			return nil
		}
		if len(val) > 1 {
			return fmt.Errorf("expected single value, got %d", len(val))
		}
		if val[0] == "Present" {
			f.Text = "Present"
			return nil
		}
		n, err := strconv.ParseInt(val[0], 10, 64)
		if err != nil {
			return err
		}
		t := time.Unix(n, 0)
		if f.Type == "date" {
			f.Text = t.Format("January 2, 2006")
		} else {
			f.Text = t.Format("January 2006")
		}
		return nil
	}

	vv := f.Value

	for i, v := range vv {
		if in(val, v.Name) {
			vv[i].Default = true
		}

		inner := vv[i].Value
		for j, v := range inner {
			if in(val, v.Name) {
				inner[j].Default = true
				vv[i].Default = true
			}
		}
	}

	return nil
}

func populateValue(ss []string) []sd.Value {
	var vals []sd.Value
	for _, s := range ss {
		vals = append(vals, sd.Value{Text: s})
	}
	return vals
}

func mapFromResource(resource interface{}) map[string]interface{} {

	m := make(map[string]interface{})

	v := reflect.ValueOf(resource)
	v = reflect.Indirect(v)
	t := reflect.TypeOf(v.Interface())

	for i := 0; i < t.NumField(); i++ {

		f := t.Field(i)

		if f.Anonymous {
			embedded := mapFromResource(v.Field(i).Interface())
			for field, value := range embedded {
				m[field] = value
			}
			continue
		}

		if _, ok := f.Tag.Lookup("ed"); ok {
			fn := strings.ToLower(f.Name)
			m[fn] = v.Field(i).Interface()
		}
	}

	return m
}
