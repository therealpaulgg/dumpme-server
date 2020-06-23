package models

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"errors"
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

	const IVSize = 16
	const bufferSize = 4096

	// we now have plaintext

	// create random initialization vector
	iv := make([]byte, IVSize)
	_, err = rand.Read(iv)
	if err != nil {
		return err
	}

	// create cipher block
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	// specify block cipher type (CTR for stream)
	ctr := cipher.NewCTR(blockCipher, iv)
	// create HMAC
	hmac := hmac.New(sha512.New, []byte(saver.SecretKey))

	buf := make([]byte, bufferSize)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		outBuf := make([]byte, n)
		ctr.XORKeyStream(outBuf, buf[:n])
		hmac.Write(outBuf)
		dest.Write(outBuf)
		if err == io.EOF {
			break
		}
	}

	dest.Write(iv)
	hmac.Write(iv)

	dest.Write(hmac.Sum(nil))
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
		// start := time.Now()

		dataStream, err := os.Open(file)

		// seek the last 16 + 64 bytes (IV length & hmac length) which contain the IV as well as HMAC
		offsetOfMetadata, err := dataStream.Seek(-1*(16+64), 2)
		if err != nil {
			return nil, err
		}

		ivAndHMAC := make([]byte, 16+64)
		_, err = dataStream.ReadAt(ivAndHMAC, offsetOfMetadata)

		if err != nil {
			return nil, err
		}

		iv := ivAndHMAC[:IVSize]
		hmacFromFile := ivAndHMAC[IVSize:]

		// create cipher block
		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		// specify block cipher type (CTR for stream)
		ctr := cipher.NewCTR(blockCipher, iv)
		// create HMAC
		hmac := hmac.New(sha512.New, []byte(saver.SecretKey))
		f, err := w.Create(file)
		if err != nil {
			return nil, err
		}

		buf := make([]byte, bufferSize)
		currentOffset := int64(0)
		dataStream.Seek(currentOffset, 0)

		// first pass
		fmt.Println("first pass")

		for {
			n, err := dataStream.ReadAt(buf, currentOffset)
			if err != nil && err != io.EOF {
				return nil, err
			}
			cryptedData := buf[:n]

			if (currentOffset+bufferSize)%offsetOfMetadata != (currentOffset + bufferSize) {
				// only include relevant bytes
				bytesToInclude := offsetOfMetadata - currentOffset
				cryptedData = cryptedData[:bytesToInclude]
			}
			hmac.Write(cryptedData)
			if err == io.EOF {
				break
			}
			currentOffset += bufferSize
		}
		hmac.Write(iv)

		currentOffset = int64(0)
		dataStream.Seek(currentOffset, 0)
		fmt.Println(hex.EncodeToString(hmacFromFile))
		fmt.Println(hex.EncodeToString(hmac.Sum(nil)))

		if bytes.Compare(hmacFromFile, hmac.Sum(nil)) != 0 {
			return nil, errors.New("authentication failed")
		}

		// second pass
		fmt.Println("second pass")

		for {
			// if we are at byte 4096 (goes to 8192) and bad byte occurs at 4112...so 4096 % 4112 != 4096
			// include bytes 4112 - 4096
			// on next iteration, we just wont include anything, easy, because math
			n, err := dataStream.ReadAt(buf, currentOffset)
			if err != nil && err != io.EOF {
				return nil, err
			}
			outBuf := make([]byte, n)
			cryptedData := buf[:n]
			var zipError error
			if (currentOffset+bufferSize)%offsetOfMetadata != (currentOffset + bufferSize) {
				// only include relevant bytes
				bytesToInclude := offsetOfMetadata - currentOffset
				cryptedData = cryptedData[:bytesToInclude]
			}

			ctr.XORKeyStream(outBuf, cryptedData)

			_, zipError = f.Write(outBuf)

			if zipError != nil {
				return nil, err
			}

			if err == io.EOF {
				break
			}

			if err != nil {
				return nil, err
			}
			currentOffset += bufferSize
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
