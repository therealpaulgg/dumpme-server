package models

import (
	"io"
	"mime/multipart"
	"os"
)

// FileSaver is a generic interface which saves a file to the persistent storage set up in user configuration.
type FileSaver interface {
	SaveFile(filename string, file multipart.File) error
}

// LocalStorageSaver implements FileSaver, uses local system storage.
type LocalStorageSaver struct {
	StoragePath string
}

// SaveFile LocalStorageSaver's implmementation of SaveFile
func (saver *LocalStorageSaver) SaveFile(filename string, file multipart.File) error {
	dest, err := os.Create(saver.StoragePath + "/" + filename)
	defer dest.Close()
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, file); err != nil {
		return err
	}
	return nil
}
