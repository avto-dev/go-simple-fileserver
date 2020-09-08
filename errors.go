package fileserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/avto-dev/go-simple-fileserver/cache"
)

// ErrorPageTemplate  is error page template in string representation. Is allowed to use basic "replacing patterns"
// like `{{ code }}` or `{{ message }}`
type ErrorPageTemplate string

// String converts template into string representation.
func (t ErrorPageTemplate) String() string { return string(t) }

// Build makes registered patterns replacing.
func (t ErrorPageTemplate) Build(errorCode int) string {
	out := t.String()

	for k, v := range map[string]string{
		"code":    strconv.Itoa(errorCode),
		"message": http.StatusText(errorCode),
	} {
		out = strings.ReplaceAll(out, fmt.Sprintf("{{ %s }}", k), v)
	}

	return out
}

type jsonError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// JSONErrorHandler respond with simple json-formatted response, if json format was requested (defined in `Accept`
// header).
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

// StaticHTMLPageErrorHandler allows to use user-defined local file with HTML for error page generating.
func StaticHTMLPageErrorHandler() ErrorHandlerFunc { //nolint:gocognit
	return func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) bool {
		if len(fs.Settings.ErrorFileName) > 0 {
			var (
				filePath        = path.Join(fs.Settings.FilesRoot, fs.Settings.ErrorFileName)
				templateContent []byte
				loaded          bool
			)

			if fs.CacheAvailable() {
				if cached, cacheHit := fs.Cache.Get(filePath); cacheHit {
					templateContent, _ = ioutil.ReadAll(cached.Content)
					loaded = true
				}
			}

			if !loaded {
				if f, err := os.Open(filePath); err == nil {
					defer f.Close()

					if data, err := ioutil.ReadAll(f); err == nil {
						templateContent = data
						loaded = true

						if fs.CacheAvailable() && fs.Cache.Count() < fs.Settings.CacheMaxItems {
							fs.Cache.Set(filePath, fs.Settings.CacheTTL, &cache.Item{
								ModifiedTime: time.Now(),
								Content:      bytes.NewReader(data),
							})
						}
					}
				}
			}

			if loaded {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(errorCode)
				_, _ = w.Write([]byte(ErrorPageTemplate(templateContent).Build(errorCode)))

				return true
			}
		}

		return false
	}
}
