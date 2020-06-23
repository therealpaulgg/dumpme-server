package filesystem

import (
	"archive/zip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

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
	// specify block cipher type (GCM)
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return err
	}

	buf := make([]byte, bufferSize)

	// 'Custom' implementation of GCM where we, for every bufferSize bytes in file, create a new block with a new nonce, and append to a binary file output.
	// The reason behind doing this is that Go's GCM implementation does not work well for large file sizes because it requires all the bytes to be in memory at once.
	// Instead, encrypt small portions at a time and reassemble into a larger file.

	for {
		n, ioErr := file.Read(buf)
		if ioErr != nil && ioErr != io.EOF {
			return ioErr
		}
		// stop on EOF
		if ioErr == io.EOF {
			break
		}

		// create nonce
		nonce := make([]byte, gcm.NonceSize())

		_, err = rand.Read(nonce)
		if err != nil {
			return err
		}

		// retrieve the encrypted block
		outBuf := gcm.Seal(nonce, nonce, buf[:n], nil)

		// write the block to the output file
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
	// find all files in the directory, ignoring the root and any pesky tempzip files
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if path != directory && !strings.HasPrefix(filepath.Base(path), "tempzip") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// create a buffer for a zip file on the filesystem (after current directories have been parsed)
	zipbuf, err := ioutil.TempFile(directory, "tempzip-")
	if err != nil {
		return nil, err
	}
	// zip file MUST be deleted once completed
	defer os.Remove(zipbuf.Name())
	// create new zipwriter
	w := zip.NewWriter(zipbuf)

	const bufferSize = 4096

	for _, file := range files {
		// obtain ciphertext as a file pointer
		dataStream, err := os.Open(file)

		// create cipher block
		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		// specify block cipher type
		gcm, err := cipher.NewGCM(blockCipher)
		if err != nil {
			return nil, err
		}

		// create ZIP file
		f, err := w.Create(file)
		if err != nil {
			return nil, err
		}

		// the encrypted blocks are larger than the normal blocks, due to prepending of nonce and additional overhead
		buf := make([]byte, bufferSize+gcm.NonceSize()+gcm.Overhead())
		for {
			n, ioErr := dataStream.Read(buf)
			if ioErr != nil && ioErr != io.EOF {
				return nil, ioErr
			}

			// on EOF, quit
			if ioErr == io.EOF {
				break
			}

			// get bytes for block
			block := buf[:n]
			// nonce is first
			nonce := block[:gcm.NonceSize()]
			// ciphertext is next
			cipherText := block[gcm.NonceSize():]

			// decrypt
			decryptedBlock, err := gcm.Open(nil, nonce, cipherText, nil)

			// most likely if error here, authentication error
			if err != nil {
				return nil, err
			}

			// write the decrypted block to the output file
			_, err = f.Write(decryptedBlock)

			if err != nil {
				return nil, err
			}
		}
	}

	// close the zip writer, all files have been written
	err = w.Close()
	if err != nil {
		return nil, err
	}
	// send zip file pointer to the main HTTP router for downloading
	return zipbuf, nil
}
