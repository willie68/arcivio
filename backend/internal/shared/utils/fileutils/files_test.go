package fileutils

import (
	"fmt"
	"io/fs"
	"testing"
)

func TestFileUtilsGetFiles(t *testing.T) {
	err := GetFiles("./", "key_", func(fileinfo fs.DirEntry) bool {
		fmt.Printf("filename: %s\r\n", fileinfo.Name())
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
}
