package setup

import (
	"fmt"
	"strconv"
)

/*
We first generate the initial mapping in a format the server
cannot actually use. While this may seem strange we do it to
maintain symmetry of generation logic between Go and Javascript
such that both can be trivially modified to produce text files
of their respective mappings that can be compared to ensure
they're the same.

Normally I'd consider this unnecesary but I think the logic
here is sufficiently baroque that having them be the same
makes for easier debugging. Plus, we're only doing this at
start-up.
*/
func timeMapping() map[string]string {

	mapped := make(map[string]string)

	// Map what we assume are common cases in most compact form.
	for i := 0; i < 10; i++ {
		h := i
		if h == 0 {
			h = 10
		}
		s := fmt.Sprintf("%02d:00pm", h)
		mapped[s] = strconv.Itoa(i)
	}

	// Map the next most common cases.
	next := 10
	for i := 1; i < 13; i++ {
		h := i
		for j := 0; j < 4; j++ {
			m := j * 15
			s := fmt.Sprintf("%02d:%02dam", h, m)
			mapped[s] = strconv.Itoa(next)
			next++
			if i > 10 || j > 0 {
				s := fmt.Sprintf("%02d:%02dpm", h, m)
				mapped[s] = strconv.Itoa(next)
				next++
			}
		}
	}

	// Map all remaining cases.
	meridiem := []string{"am", "pm"}
	leadingZeroes := 0
	seen100 := false
	seen10 := false
	for i := 1; i < 13; i++ {
		for j := 1; j < 60; j++ {
			if j%15 == 0 {
				continue
			}
			for _, ap := range meridiem {
				if !seen100 && next == 100 {
					next = -9
					leadingZeroes = 2
					seen100 = true
				}
				if !seen10 && next == 10 {
					next = -99
					leadingZeroes = 3
					seen10 = true
				}
				z := strconv.Itoa(leadingZeroes)
				s := fmt.Sprintf("%02d:%02d%s", i, j, ap)
				mapped[s] = fmt.Sprintf("%0"+z+"d", next)
				next++
			}
		}
	}

	/*
		Then we transform the map into one that can
		receive compact values and decompress them.
	*/
	return timeMapToServer(mapped)
}

func timeMapToServer(clientMapping map[string]string) map[string]string {

	serverMapping := make(map[string]string)

	for k, v := range clientMapping {

		h, err := strconv.Atoi(k[0:2])
		if err != nil {
			panic(err)
		}
		m, err := strconv.Atoi(k[3:5])
		if err != nil {
			panic(err)
		}
		isPM := k[5:7] == "pm"

		if h < 12 && isPM {
			h += 12
		}
		if h == 12 && !isPM {
			h = 0
		}

		n := ((h * 60) * 60) + (m * 60)

		serverMapping[v] = strconv.Itoa(n)
	}

	return serverMapping
}
