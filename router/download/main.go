package download

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/therealpaulgg/dumpme-server/services"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Get("/{foldername}/{key}", download)
	return r
}

func download(w http.ResponseWriter, req *http.Request) {
	foldername := chi.URLParam(req, "foldername")
	encodedKey := chi.URLParam(req, "key")
	key, err := base64.URLEncoding.DecodeString(encodedKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
	}
	zip, err := services.EncryptedFileSaver.GetFiles(foldername, key)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	defer os.Remove(zip.Name())
	zip.Seek(0, 0)

	w.Header().Set("Content-Disposition", "attachment; filename=decrypted.zip")
	w.Header().Set("Content-Type", "application/zip")
	// use a larger buffer to copy
	for {
		buf := make([]byte, 4096)
		_, readErr := zip.Read(buf)
		if readErr != nil && readErr != io.EOF {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
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

	// zip.Close()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
}
