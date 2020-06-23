package download

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/therealpaulgg/dumpme-server/services"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Get("/{foldername}/{key}", download)
	return r
}

func download(w http.ResponseWriter, req *http.Request) {
	// foldername and key are part of URL
	foldername := chi.URLParam(req, "foldername")
	encodedKey := chi.URLParam(req, "key")
	// key should be encoded in URLBase64
	key, err := base64.URLEncoding.DecodeString(encodedKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
	}
	// Attempt to decrypt and create a ZIP of relevant files
	zip, err := services.EncryptedFileSaver.GetFiles(foldername, key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	// reset ZIP pointer to beginning for reading, it was just written to and is at EOF unless we do this
	zip.Seek(0, 0)

	w.Header().Set("Content-Disposition", "attachment; filename=decrypted.zip")
	w.Header().Set("Content-Type", "application/zip")
	// use a larger buffer to copy, for some reason it has better performance than io.Copy
	for {
		buf := make([]byte, 4096)
		_, readErr := zip.Read(buf)
		if readErr != nil && readErr != io.EOF {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		// this writes the bytes to the user downloading
		_, err := w.Write(buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		if readErr == io.EOF {
			break
		}
	}
}
