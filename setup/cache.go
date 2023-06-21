package setup

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/internal/cache"
)

func mustCache(c *sd.Config) sd.Cache {
	cache := cache.New()
	cache.OnLoad(onSVGLoad)
	mustPopulateCache(c, cache)
	return cache
}

func mustPopulateCache(c *sd.Config, cache sd.Cache) {
	cache.MustAddFile("favicon.ico", c.PathFavIcon)
	cache.MustConcatDir("js/init.js", c.DirJSInit, nil, true)
	cache.MustConcatDir("js/script.js", c.DirJS, []string{c.DirJSInit}, true)
	cache.MustConcatDir("css/styling.css", c.DirCSS, nil, false)
	cache.MustAddDir("fonts", c.DirFonts, nil, true)
	cache.MustAddDir("gfx", c.DirGFX, nil, false)
	cache.MustAddDir("svg", c.DirSVG, nil, true)
	cache.MustAddDir("ui/mode", c.DirMode, nil, false)
	cache.MustAddFile("robots.txt", c.PathRobots)
}

func onSVGLoad(ext string, file []byte) (modified []byte) {

	if ext != ".svg" {
		return file
	}

	svg := string(file)
	svgParts := strings.SplitAfter(svg, ">")
	svg = ""

	re := regexp.MustCompile(`^</?(\?xml|!--|defs|clipPath|g)( |>)`)
	fill := regexp.MustCompile(` fill="rgb\(\d+,\d+,\d+\)"`)
	style := regexp.MustCompile(` style="stop-color:rgb\(\d+,\d+,\d+\)"`)

	for _, p := range svgParts {

		if re.MatchString(p) {
			continue
		}

		if strings.HasPrefix(p, "<rect width") {
			continue
		}

		if strings.HasPrefix(p, "<svg ") {

			var viewBox string
			attrs := strings.SplitAfter(strings.TrimSuffix(p, ">"), `" `)
			p = "<svg "

			for _, attr := range attrs {
				if strings.HasPrefix(attr, "viewBox") {
					viewBox = strings.TrimSpace(attr)
					p += viewBox
					break
				}
			}

			dimensions := strings.Split(viewBox, " ")
			w, err := strconv.Atoi(dimensions[2])
			if err != nil {
				panic(err)
			}
			h, err := strconv.Atoi(strings.TrimRight(dimensions[3], `"`))
			if err != nil {
				panic(err)
			}

			p += fmt.Sprintf(` style="width: %dpx; height: %dpx;">`, w/10, h/10)
		}

		p = fill.ReplaceAllString(p, "")
		p = style.ReplaceAllString(p, "")

		svg += p
	}

	return []byte(svg)
}
