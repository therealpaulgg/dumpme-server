package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zip"
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
	SecretKey   string
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

	const bufferSize = 4096

	// we now have plaintext

	// create cipher block
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	// specify block cipher type (CTR for stream)
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return err
	}

	buf := make([]byte, bufferSize)

	for {
		n, ioErr := file.Read(buf)
		fmt.Println(n)
		if ioErr != nil && ioErr != io.EOF {
			return ioErr
		}
		nonce := make([]byte, gcm.NonceSize())

		_, err = rand.Read(nonce)
		if err != nil {
			return err
		}
		if ioErr == io.EOF {
			break
		}

		outBuf := gcm.Seal(nonce, nonce, buf[:n], nil)
		fmt.Println(len(outBuf))

		dest.Write(outBuf)

	}
	dest.Close()

	return nil
}

// GetFiles decrypts all files in directory and zips them up
func (saver *LocalStorageSaverAES) GetFiles(foldername string, key []byte) (*os.File, error) {
	directory := saver.StoragePath + "/" + foldername
	if _, err := os.Stat(directory); err != nil {
		return nil, err
	}

	var files []string
	// find all files in the directory, ignoring the root
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if path != directory && !strings.HasPrefix(filepath.Base(path), "tempzip") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// create a buffer for a zip file in memory' (after current directories have been parsed)
	zipbuf, err := ioutil.TempFile(directory, "tempzip-")
	zipbuf.Name()
	if err != nil {
		return nil, err
	}
	w := zip.NewWriter(zipbuf)

	const IVSize = 16
	const bufferSize = 4096

	for _, file := range files {
		// obtain ciphertext and add the decrypted version to an archive

		dataStream, err := os.Open(file)

		// create cipher block
		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		// specify block cipher type (CTR for stream)
		gcm, err := cipher.NewGCM(blockCipher)
		if err != nil {
			return nil, err
		}

		// create ZIP file
		f, err := w.Create(file)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, bufferSize+gcm.NonceSize()+gcm.Overhead())
		for {
			n, ioErr := dataStream.Read(buf)
			if ioErr != nil && ioErr != io.EOF {
				return nil, ioErr
			}

			if ioErr == io.EOF {

				break
			}

			block := buf[:n]
			nonce := block[:gcm.NonceSize()]
			cipherText := block[gcm.NonceSize():]

			decryptedBlock, err := gcm.Open(nil, nonce, cipherText, nil)

			if err != nil {
				return nil, err
			}

			_, err = f.Write(decryptedBlock)

			if err != nil {
				return nil, err
			}

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
