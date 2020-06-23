package upload

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	start := time.Now()
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	fmt.Println("Just finished parsing multipart", time.Since(start))
	m := req.MultipartForm
	files := m.File["files"]
	foldername, _ := gonanoid.Nanoid(54)
	// 256 bit AES is 32 bytes
	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
	}
	// key, _ := gonanoid.Nanoid(32)
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
		fmt.Println("Added & encrypted file", time.Since(start))
	}
	encodedKey := base64.URLEncoding.EncodeToString(key)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folderName": foldername,
		"key":        encodedKey,
	})
}
