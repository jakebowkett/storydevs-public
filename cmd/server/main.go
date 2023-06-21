package main

import (
	"net/http"
	"time"

	"github.com/jakebowkett/storydevs/setup"
)

func main() {

	path := "./config.default.toml"
	dep, handles := setup.Dependencies(path)
	defer handles.Close()

	c := dep.Config
	log := dep.Logger

	rt := setup.MustRoutes(dep)
	s := http.Server{
		Handler:      rt,
		WriteTimeout: time.Second * 20,
		ReadTimeout:  time.Second * 20,
		IdleTimeout:  time.Second * 60,
	}

	s.Addr = ":" + c.Port
	log.OnceF("StoryDevs server listening on port %s.", c.Port)
	log.Fatal(s.ListenAndServe())
}
