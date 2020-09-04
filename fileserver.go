package fileserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// this HTML content will be used as fallback content for 404 response.
const fallback404content = "<html><body><h1>ERROR 404</h1><h2>Requested file was not found</h2></body></html>"

type json404error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type (
	// FileNotFoundHandler handle requests, when requested file was not found on server
	FileNotFoundHandler func(*FileServer, http.ResponseWriter, *http.Request) (makeStop bool)

	Settings struct {
		Root         http.Dir
		IndexFile    string
		Error404file string
	}

	FileServer struct {
		Settings         Settings
		NotFoundHandlers []FileNotFoundHandler // optional
	}
)

// NewFileServer creates new file server.
func NewFileServer(settings Settings) *FileServer {
	return &FileServer{
		Settings: settings,
		NotFoundHandlers: []FileNotFoundHandler{
			JSON404errorHandler(),
			StaticHtmlPage404errorHandler(),
		},
	}
}

// Serve requests to the "public" files and directories.
func (fileServer *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// redirect .../index.html to .../
	if strings.HasSuffix(r.URL.Path, "/"+fileServer.Settings.IndexFile) {
		http.Redirect(w, r, r.URL.Path[0:len(r.URL.Path)-len(fileServer.Settings.IndexFile)], http.StatusMovedPermanently)
		return
	}

	// if empty, set current directory
	dir := string(fileServer.Settings.Root)
	if dir == "" {
		dir = "."
	}

	// add prefix and clean
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
		r.URL.Path = upath
	}
	// add index file name if requested directory (or server root)
	if upath[len(upath)-1] == '/' {
		upath += fileServer.Settings.IndexFile
	}
	// make path clean
	upath = path.Clean(upath)

	// path to file
	name := path.Join(dir, filepath.FromSlash(upath))

	// if files server root directory is set - try to find file and serve them
	if len(fileServer.Settings.Root) > 0 {
		// check for file exists
		if f, err := os.Open(name); err == nil {
			// file exists and opened
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()
			// file (or directory) exists
			if stat, statErr := os.Stat(name); statErr == nil && stat.Mode().IsRegular() {
				// requested file is file (not directory)
				var modTime time.Time
				// Try to extract file modified time
				if info, err := f.Stat(); err == nil {
					modTime = info.ModTime()
				} else {
					modTime = time.Now() // fallback
				}
				// serve fie content
				http.ServeContent(w, r, filepath.Base(upath), modTime, f)

				return
			}
		}
	}

	fileServer.handle404(w, r)
}

func (fileServer *FileServer) handle404(w http.ResponseWriter, r *http.Request) {
	// If all tries for content serving above has been failed - file was not found (HTTP 404)
	if fileServer.NotFoundHandlers != nil || len(fileServer.NotFoundHandlers) == 0 {
		for _, handler := range fileServer.NotFoundHandlers {
			if handler(fileServer, w, r) {
				return
			}
		}
	}

	// fallback
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)

	_, _ = fmt.Fprint(w, fallback404content)
}

func JSON404errorHandler() FileNotFoundHandler {
	return func(fs *FileServer, w http.ResponseWriter, r *http.Request) bool {
		if strings.Contains(r.Header.Get("Accept"), "json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)

			_ = json.NewEncoder(w).Encode(json404error{
				Code:    http.StatusNotFound,
				Message: "Not found",
			})

			return true
		}

		return false
	}
}

func StaticHtmlPage404errorHandler() FileNotFoundHandler {
	return func(fs *FileServer, w http.ResponseWriter, r *http.Request) bool {
		if len(fs.Settings.Root) > 0 {
			var errPage = path.Join(string(fs.Settings.Root), fs.Settings.Error404file)
			if f, err := os.Open(errPage); err == nil {
				// file exists and opened
				defer func() {
					if err := f.Close(); err != nil {
						panic(err)
					}
				}()

				// file (or directory) exists
				if stat, statErr := os.Stat(errPage); statErr == nil && stat.Mode().IsRegular() {
					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					w.WriteHeader(http.StatusNotFound)

					// requested file is file (not directory)
					if _, writeErr := io.Copy(w, f); writeErr != nil {
						panic(writeErr)
					}

					return true
				}
			}
		}

		return false
	}
}
