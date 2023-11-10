package util

import (
	"os"
)

var (
	TmpDir        string = "/tmp"
	TmpFilePatter string = "probe-"
)

// TmpFileName creates a file with temporary name in TmpDir
// probeFactory the dir is writable
// It is the caller's responsibility to remove the file when no longer needed.
func TmpFileName() (name string, err error) {
	if f, err := TmpFile(); err != nil {
		return "", err
	} else {
		_ = f.Close()
		return f.Name(), nil
	}
}

func TmpFile() (file *os.File, err error) {
	return os.CreateTemp(TmpDir, TmpFilePatter)
}

func StringToFile(s string) (name string, err error) {
	f, err := TmpFile()
	if err != nil {
		return
	}

	name = f.Name()

	defer func() {
		_ = f.Close()
	}()

	_, err = f.WriteString(s)

	return
}
