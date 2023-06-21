package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	L "github.com/jakebowkett/go-logger/logger"
	sd "github.com/jakebowkett/storydevs"
)

func logger(c *sd.Config) (sd.Logger, *os.File) {
	logEvent, f := onLog(c)
	log := &L.Logger{OnLog: logEvent}
	log.SetDebug(c.Debug)
	log.SetRuntime(c.RuntimeLogging)
	return log, f
}

func onLog(c *sd.Config) (func(L.Thread), *os.File) {

	var mu sync.Mutex

	// Path to logs directory.
	path, err := filepath.Abs(c.DirLogs)
	if err != nil {
		panic(err)
	}

	// Make path to file.
	path = filepath.Join(path, "log.txt")

	// If file doesn't exist, create it, or append to it.
	flags := os.O_APPEND | os.O_CREATE | os.O_WRONLY
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		panic(err)
	}

	return func(t L.Thread) {

		if c.Console {
			fmt.Print(t.FormatPretty())
		}

		s := t.FormatRecord()
		mu.Lock()
		f.WriteString(s)
		mu.Unlock()
	}, f
}
