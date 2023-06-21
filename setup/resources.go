package setup

import (
	"strings"

	sd "github.com/jakebowkett/storydevs"
)

func mapResources(vd *sd.ViewData) (mp sd.ResourceMapping) {

	mp = make(sd.ResourceMapping)

	for _, md := range vd.Mode {
		mp[md.Name] = mappingResources(sd.Fields(md.Editor), false)
	}
	for _, md := range vd.Modal {
		mp[md.Name] = mappingResources(sd.Fields(md.Field), true)
	}

	return mp
}

func mappingResources(ff sd.Fields, modal bool) map[string]string {

	m := &mapper{
		mapping: make(map[string]string),
	}

	for _, f := range ff {

		m.pushSource(f.Name)
		if modal || f.Add > 0 {
			m.pushResource(f.Name)
			m.commitMapping()
		}

		m.mapFields(f.Field)

		m.popSource()
		if modal || f.Add > 0 {
			m.popResource()
		}
	}

	return m.mapping
}

type chain []string

func (c chain) String() string {
	return strings.Join(c, ".")
}

type mapper struct {
	source   chain
	resource chain
	mapping  map[string]string
}

func (m *mapper) pushSource(s string) {
	m.source = append(m.source, s)
}
func (m *mapper) popSource() {
	m.source = m.source[0 : len(m.source)-1]
}
func (m *mapper) pushResource(s string) {
	m.resource = append(m.resource, s)
}
func (m *mapper) popResource() {
	m.resource = m.resource[0 : len(m.resource)-1]
}
func (m *mapper) commitMapping() {
	m.mapping[m.resource.String()] = m.source.String()
}
func (m *mapper) mapFields(ff []sd.Field) {
	for _, f := range ff {
		m.pushSource(f.Name)
		/*
			Sometimes we skip names of fields. But we always
			push names if they're an adder, a leaf field, or
			if they have been specified in the TOML file to
			be included.
		*/
		if f.Add > 0 || len(f.Field) == 0 || f.SubmitSingle {
			m.pushResource(f.Name)
		}
		m.commitMapping()
		m.mapFields(f.Field)
		m.popSource()
		if f.Add > 0 || len(f.Field) == 0 || f.SubmitSingle {
			m.popResource()
		}
	}
}
