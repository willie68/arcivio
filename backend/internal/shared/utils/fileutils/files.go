package fileutils

import (
	"io/fs"
	"os"
	"strings"
)

// Callback reports whether GetFiles should keep reading. false stops the walk.
type Callback func(fileinfo fs.DirEntry) bool

// GetFiles reads rootpath and calls callback for each entry whose name does not start with prefix.
func GetFiles(rootpath, prefix string, callback Callback) error {
	infos, err := os.ReadDir(rootpath)
	if err != nil {
		return err
	}
	for _, i := range infos {
		if !strings.HasPrefix(i.Name(), prefix) {
			ok := callback(i)
			if !ok {
				return nil
			}
		}
	}
	return nil
}
