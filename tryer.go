package storydevs

import (
	"errors"
	"os"
)

type Tryer interface {
	Try(fn func() error) (errs []error, err error)
}

type RetryError error

var (
	ErrRetryTx    = errors.New("")
	ErrRetryEmail = errors.New("")
	ErrRetryDisk  = errors.New("file with this name already exists")
)

func RetryDefault(err error) bool {
	return true
}

func RetryDisk(err error) bool {
	return os.IsExist(err)
}
