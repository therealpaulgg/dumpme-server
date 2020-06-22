package models

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
)

// FileSaver is a generic interface which saves a file to the persistent storage set up in user configuration.
type FileSaver interface {
	SaveFile(filename string, foldername string, file multipart.File) error
}

// EncryptedFileSaver is a generic interface which saves a file to the persistent storage set up in user configuration.
type EncryptedFileSaver interface {
	SaveFile(filename string, foldername string, key []byte, file multipart.File) error
	GetFiles(foldername string, key []byte) (*bytes.Buffer, error)
}

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

// LocalStorageSaverAES implements FileSaver, uses local system storage. Makes use of AES.
type LocalStorageSaverAES struct {
	StoragePath string
}

// SaveFile LocalStorageSaver's implmementation of SaveFile
func (saver *LocalStorageSaverAES) SaveFile(filename string, foldername string, key []byte, file multipart.File) error {
	// Check if directory exists, create if it doesn't exist yet
	_, err := os.Stat(saver.StoragePath + "/" + foldername)
	if os.IsNotExist(err) {
		os.Mkdir(saver.StoragePath+"/"+foldername, 0755)
	} else if err != nil {
		return err
	}

	// Create the location where the encrypted file will go
	dest, err := os.Create(saver.StoragePath + "/" + foldername + "/" + filename)
	if err != nil {
		return err
	}
	defer dest.Close()

	// convert multipart file to bytes
	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, file); err != nil {
		return err
	}

	plaintext := buf.Bytes()

	// we now have plaintext

	// create cipher block
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	// specify block cipher type (GCM, authenticated)
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return err
	}
	// create a random nonce
	nonce := make([]byte, gcm.NonceSize())

	if _, err = rand.Read(nonce); err != nil {
		return err
	}
	// create and write ciphertext with nonce as first 12 bytes
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	if _, err := dest.Write(ciphertext); err != nil {
		return err
	}

	return nil
}

// GetFiles decrypts all files in directory and zips them up
func (saver *LocalStorageSaverAES) GetFiles(foldername string, key []byte) (*bytes.Buffer, error) {
	directory := saver.StoragePath + "/" + foldername
	if _, err := os.Stat(directory); err != nil {
		return nil, err
	}
	// create a buffer for a zip file in memory
	zipbuf := new(bytes.Buffer)
	w := zip.NewWriter(zipbuf)

	var files []string
	// find all files in the directory, ignoring the root
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if path != directory {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// obtain ciphertext and add the decrypted version to an archive
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		gcm, err := cipher.NewGCM(blockCipher)
		if err != nil {
			return nil, err
		}
		// collect nonce and ciphertext separately from data
		nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
		// attempt decryption
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			// decrypton failed
			return nil, err
		}

		// create a file with the decrypted file inside of the ZIP file
		f, err := w.Create(file)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(plaintext)
		if err != nil {
			return nil, err
		}
	}
	// close the zip writer
	err = w.Close()
	if err != nil {
		return nil, err
	}
	// send zip file buffer
	return zipbuf, nil
}
