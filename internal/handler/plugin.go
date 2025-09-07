package handler

import (
	"net/http"
	"os"
)

func HandlePluginEntry(res http.ResponseWriter, req *http.Request) {
	entryFilePath := req.URL.Query().Get("path")
	if _, err := os.Stat(entryFilePath); os.IsNotExist(err) {
		http.NotFound(res, req)
		return
	}
	http.ServeFile(res, req, entryFilePath)
}
