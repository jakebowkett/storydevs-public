package storydevs

import "github.com/jakebowkett/storydevs/internal/cache"

const CachePrivate = "private"
const CachePublic = "public"

type Cache interface {
	List() (aliases []string)

	AddString(alias, s string)

	MustConcatDir(alias, dirPath string, exts []string, recursive bool)
	MustAddDir(alias, dirPath string, exts []string, recursive bool)
	MustAddFile(alias, filePath string)

	ConcatDir(alias, dirPath string, exts []string, recursive bool) error
	AddDir(alias, dirPath string, exts []string, recursive bool) error
	AddFile(alias, filePath string) error

	Delete(alias string)
	Load(alias string) *cache.Object
	LoadDir(alias string) []*cache.Object

	Refresh() (dropped []string)

	Empty()
}
