package filesystem

import (
	"io"
	"mime/multipart"
	"os"
)

// LocalStorageSaver implements FileSaver, uses local system storage.
type LocalStorageSaver struct {
	StoragePath string
}

// SaveFile LocalStorageSaver's implmementation of SaveFile
func (saver *LocalStorageSaver) SaveFile(filename string, foldername string, file multipart.File) error {
	_, err := os.Stat(saver.StoragePath + "/" + foldername)
	if os.IsNotExist(err) {
		os.Mkdir(saver.StoragePath+"/"+foldername, 0755)
	} else if err != nil {
		return err
	}
	dest, err := os.Create(saver.StoragePath + "/" + foldername + "/" + filename)
	defer dest.Close()
	if err != nil {
		return err
	}
	if _, err := io.Copy(dest, file); err != nil {
		return err
	}
	return nil
}
