package download

import (
	"encoding/base64"
	"encoding/json"
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
	}
	w.Header().Set("Content-Disposition", "attachment; filename=decrypted.zip")
	w.Header().Set("Content-Type", "application/zip")
	w.Write(zip.Bytes())
}
