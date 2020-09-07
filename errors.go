package fileserver

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
)

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func JSONErrorHandler() ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) bool {
		if strings.Contains(r.Header.Get("Accept"), "json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(errorCode)

			_ = json.NewEncoder(w).Encode(jsonError{
				Code:    errorCode,
				Message: http.StatusText(errorCode),
			})

			return true
		}

		return false
	}
}

func StaticHtmlPageErrorHandler() ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) bool {
		if len(fs.Settings.ErrorFileName) > 0 {
			if f, err := os.Open(path.Join(fs.Settings.FilesRoot, fs.Settings.ErrorFileName)); err == nil {
				defer f.Close()

				// TODO: add error page content caching

				if data, err := ioutil.ReadAll(f); err == nil {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(errorCode)

					_, _ = w.Write([]byte(PrepareErrorContent(string(data), errorCode)))

					return true
				}
			}
		}

		return false
	}
}
