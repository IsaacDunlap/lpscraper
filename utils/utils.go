package utils

import (
	"os"
	"path/filepath"
)

// Exit is a struct used to exit whilst respecting deferred function calls.
// Instead of calling os.Exit(code), call panic with Exit{Code: code} as the
// variable. This will trigger an exit.
type Exit struct {
	Code int
}

func HandleExit() {
	if e := recover(); e != nil {
		if exit, ok := e.(Exit); ok {
			os.Exit(exit.Code)
		}
		panic(e) // not an Exit, bubble up
	}
}

// TODO fix bugs - doesn't work
func RemoveContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
