package models

import (
	"mime/multipart"
	"os"
)

// FileSaver is a generic interface which saves a file to the persistent storage set up in user configuration.
type FileSaver interface {
	SaveFile(filename string, foldername string, file multipart.File) error
}

// EncryptedFileSaver is a generic interface which saves a file to the persistent storage set up in user configuration.
type EncryptedFileSaver interface {
	SaveFile(filename string, foldername string, key []byte, file multipart.File) error
	GetFiles(foldername string, key []byte) (*os.File, error)
}
