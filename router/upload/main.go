package upload

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/therealpaulgg/dumpme-server/services"
)

// Router return upload router
func Router() chi.Router {
	r := chi.NewRouter()
	r.Post("/", uploadFiles)
	return r
}

func uploadFiles(w http.ResponseWriter, req *http.Request) {
	// When creating a file, use 10 << 20 (10 megabytes) in memory, otherwise, create temporary files
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	m := req.MultipartForm
	files := m.File["files"]
	// create a random nanoid for the foldername
	foldername, _ := gonanoid.Nanoid(54)
	// 256 bit AES is 32 bytes
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
	}
	// try to write all files to the system. this should not fail.
	for i := range files {
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		err = services.EncryptedFileSaver.SaveFile(files[i].Filename, foldername, key, file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
	}
	// base64 encode the AES key, and include in the URL. without this, no way to recover the uploaded data.
	// USING THIS PROGRAM OVER HTTP IS INHERENTLY INSECURE AS THE BASE64 URL ENCODED KEY WILL BE VISIBLE IN PLAINTEXT.
	encodedKey := base64.URLEncoding.EncodeToString(key)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"folderName": foldername,
		"key":        encodedKey,
		// generate a convenient URL
		"url": fmt.Sprintf("%s://%s/download/%s/%s", "http", req.Host, foldername, encodedKey),
	})
}
