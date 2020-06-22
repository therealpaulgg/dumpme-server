package upload

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/therealpaulgg/dumpme-server/services"
)

// Router return upload router
func Router() chi.Router {
	r := chi.NewRouter()
	r.Post("/", uploadFiles)
	return r
}

func uploadFiles(w http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	m := req.MultipartForm
	files := m.File["files"]
	for i := range files {
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = services.FileSaver.SaveFile(files[i].Filename, file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
