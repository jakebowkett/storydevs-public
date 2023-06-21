package storydevs

import (
	"errors"
	"time"
)

type days struct {
	time.Duration
}

func (d *days) UnmarshalTOML(i interface{}) error {
	n, ok := i.(int64)
	if !ok {
		return errors.New("expected int64 when unmarshalling to type days")
	}
	(*d).Duration = time.Hour * 24 * time.Duration(n)
	return nil
}

type Config struct {
	MaxPersonas int

	SiteDesc    string
	SiteCardURL string
	SiteCardAlt string
	SiteTwitter string

	UpdateDelta int

	Dev            bool
	Console        bool
	Debug          bool
	PrettyLogging  bool
	RuntimeLogging bool
	Cache          bool

	CacheControl bool

	CacheHTML      days
	CacheSiteFiles days
	CacheUserFiles days
	CacheFavicon   days
	CacheRobots    days

	// Empty text for columns.
	Empty map[string]string

	MaxViewRequest int64

	//
	ClampImageKiB int
	ThumbMaxAxis  int

	// These are measured in bytes.
	MaxForm map[string]int64

	// User-facing hints on how to use particular
	// input types in the search and editor UI.
	InputHelp map[string]string

	// Mode specifc config.
	Thread ThreadConfig

	SlugLen int

	URIReserved string

	MinHandle int
	MaxHandle int
	MinEmail  int
	MaxEmail  int
	MinPass   int
	MaxPass   int

	DirLogs      string
	DirStore     string
	DirJS        string
	DirJSInit    string
	DirFonts     string
	DirCSS       string
	DirGFX       string
	DirUser      string
	DirSVG       string
	DirIcons     string
	DirTemplates string
	DirMode      string
	DirModal     string
	DirPage      string
	DirError     string
	DirShared    string
	DirReplace   string

	PathConfigLocal     string
	PathCredentials     string
	PathRobots          string
	PathFavIcon         string
	PathHyphenate       string
	PathHyphenateCustom string
	PathDbTypes         string
	PathDbTables        string
	PathTimezone        string

	BcryptCost int

	Port string

	Retry map[string]RetryConfig

	EmailEnabled bool
	EmailTimeout int
	EmailOnError []string

	Credentials Credentials
}

type Credentials struct {
	InitHandle string
	InitEmail  string
	InitPass   string
	EmailHost  string
	EmailPort  string
	EmailUser  string
	EmailPass  string
	EmailFrom  string
	EmailName  string
	DbConn     string
}

type RetryConfig struct {
	Retries  int
	Exponent float64
	Jitter   float64

	// In milliseconds.
	Base        int
	MaxInterval int
	MaxWait     int
}

type ThreadConfig struct {
	MinTitle         int
	MaxTitle         int
	MaxSummary       int
	MinBody          int
	MaxBody          int
	MaxParagraph     int
	MinCategoryCount int
	MaxCategoryCount int
	MinTag           int
	MaxTag           int
	MinTagCount      int
	MaxTagCount      int
	MaxComment       int
	ParagraphTypes   []string
	InlineStyles     []string
}
