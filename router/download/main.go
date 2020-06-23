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
	zip.Seek(0, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=decrypted.zip")
	w.Header().Set("Content-Type", "application/zip")
	_, err = io.Copy(w, zip)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	os.Remove(zip.Name())
}
