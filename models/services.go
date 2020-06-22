package models

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
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
	// ensure folder name exists
	_, err := os.Stat(saver.StoragePath + "/" + foldername)
	if os.IsNotExist(err) {
		os.Mkdir(saver.StoragePath+"/"+foldername, 0755)
	} else if err != nil {
		return err
	}
	// this is the outfile name
	outFilename := filename + ".enc"
	dest, err := os.Create(saver.StoragePath + "/" + foldername + "/" + outFilename)
	if err != nil {
		return err
	}
	defer dest.Close()

	size, _ := file.Seek(0, 0)
	origSize := uint64(size)
	if err = binary.Write(dest, binary.LittleEndian, origSize); err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)

	if _, err := io.Copy(buf, file); err != nil {
		return err
	}

	plaintext := buf.Bytes()

	// we now have plaintext

	if len(plaintext)%aes.BlockSize != 0 {
		bytesToPad := aes.BlockSize - (len(plaintext) % aes.BlockSize)
		padding := make([]byte, bytesToPad)
		if _, err := rand.Read(padding); err != nil {
			return err
		}
		plaintext = append(plaintext, padding...)
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return err
	}
	if _, err = dest.Write(iv); err != nil {
		return err
	}

	// ciphertext := make([]byte, len(plaintext))
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	mode, err := cipher.NewGCM(block)
	nonce := make([]byte, 12)
	ciphertext := mode.Seal(nil, nonce, plaintext, nil)
	// mode.CryptBlocks(ciphertext, plaintext)

	if _, err := dest.Write(ciphertext); err != nil {
		return err
	}

	return nil
}

func (saver *LocalStorageSaverAES) GetFiles(foldername string, key []byte) (*bytes.Buffer, error) {
	zipbuf := new(bytes.Buffer)
	w := zip.NewWriter(zipbuf)
	directory := saver.StoragePath + "/" + foldername
	fmt.Println(directory)
	var files []string
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
		ciphertext, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, err
		}
		var origSize uint64
		buf := bytes.NewReader(ciphertext)
		if err = binary.Read(buf, binary.LittleEndian, &origSize); err != nil {
			return nil, err
		}
		iv := make([]byte, aes.BlockSize)
		if _, err := buf.Read(iv); err != nil {
			return nil, err
		}

		paddedSize := len(ciphertext) - 8 - aes.BlockSize
		if paddedSize%aes.BlockSize != 0 {
			return nil, fmt.Errorf("want padded plaintext size to be aligned to block size")
		}
		plaintext := make([]byte, paddedSize)

		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		mode := cipher.NewCBCDecrypter(block, iv)
		mode.CryptBlocks(plaintext, ciphertext[8+aes.BlockSize:])
		f, err := w.Create(file)
		if err != nil {
			return nil, err
		}
		_, err = f.Write(plaintext)
		if err != nil {
			return nil, err
		}
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return zipbuf, nil
}
