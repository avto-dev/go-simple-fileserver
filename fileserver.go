package fileserver

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/avto-dev/go-simple-fileserver/cache"
)

const (
	defaultFallbackErrorContent = "<html><body><h1>Error {{ code }}</h1><h2>{{ message }}</h2></body></html>"
	defaultIndexFileName        = "index.html"
	defaultCacheTTL             = time.Second * 5
	defaultCacheMaxFileSize     = 1024 * 64 // 64 KiB
	defaultCacheMaxItems        = 64
)

type FileServer struct {
	Settings             Settings
	Cache                cache.Cacher // nil, if caching disabled
	FallbackErrorContent string
}

type Settings struct {
	// Directory path, where files for serving is located.
	FilesRoot string

	// File name (relative path to the file) that will be used as an index (like <https://bit.ly/356QeFm>).
	IndexFileName string

	// File name (relative path to the file) that will be used as error page template.
	ErrorFileName string

	// Respond direct index file request with redirection to root (`example.com/index.html` -> `example.com/`).
	RedirectIndexFileToRoot bool

	// Enables caching engine.
	CacheEnabled bool

	// Maximal data caching lifetime.
	CacheTTL time.Duration

	// Maximum file size (in bytes), that can be placed into the cache.
	CacheMaxFileSize int64

	// Maximum files count, that can be placed into the cache.
	CacheMaxItems uint32
}

func NewFileServer(s Settings) (*FileServer, error) {
	if info, err := os.Stat(s.FilesRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(`directory "%s" does not exists`, s.FilesRoot)
		}
	} else if !info.IsDir() {
		return nil, fmt.Errorf(`"%s" is not directory`, s.FilesRoot)
	}

	if s.IndexFileName == "" {
		s.IndexFileName = defaultIndexFileName
	}

	if s.CacheTTL == 0 {
		s.CacheTTL = defaultCacheTTL
	}

	if s.CacheMaxFileSize == 0 {
		s.CacheMaxFileSize = defaultCacheMaxFileSize
	}

	if s.CacheMaxItems == 0 {
		s.CacheMaxItems = defaultCacheMaxItems
	}

	fs := &FileServer{
		Settings:             s,
		FallbackErrorContent: defaultFallbackErrorContent,
	}

	if s.CacheEnabled {
		fs.Cache = cache.NewInMemoryCache(s.CacheTTL / 2)
	}

	return fs, nil
}

func (fs *FileServer) cacheAvailable() bool {
	return fs.Settings.CacheEnabled && fs.Cache != nil
}

func (fs *FileServer) handleError(errorCode int, w http.ResponseWriter, r *http.Request) {
	//
}

func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if fs.Settings.RedirectIndexFileToRoot && len(fs.Settings.IndexFileName) > 0 {
		// redirect .../index.html to .../
		if strings.HasSuffix(r.URL.Path, "/"+fs.Settings.IndexFileName) {
			http.Redirect(w, r, r.URL.Path[0:len(r.URL.Path)-len(fs.Settings.IndexFileName)], http.StatusMovedPermanently)

			return
		}
	}

	urlPath := r.URL.Path

	// add leading `/` (if required)
	if len(urlPath) == 0 || !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + r.URL.Path
	}

	// if directory requested (or server root) - add index file name
	if urlPath[len(urlPath)-1] == '/' {
		urlPath += fs.Settings.IndexFileName
	}

	// prepare target file path
	filePath := path.Join(fs.Settings.FilesRoot, filepath.FromSlash(path.Clean(urlPath)))

	// look for response in cache
	if fs.cacheAvailable() {
		if cached, cacheHit := fs.Cache.Get(filePath); cacheHit {
			http.ServeContent(w, r, filepath.Base(filePath), cached.ModifiedTime, cached.Content)

			return
		}
	}

	// check for file existence
	if stat, err := os.Stat(filePath); err == nil && stat.Mode().IsRegular() {
		if file, err := os.Open(filePath); err == nil {
			defer file.Close()

			var fileContent io.ReadSeeker

			// put file content into cache, if possible
			if fs.cacheAvailable() &&
				fs.Cache.Count() < fs.Settings.CacheMaxItems &&
				stat.Size() > 0 &&
				stat.Size() <= fs.Settings.CacheMaxFileSize {
				if data, err := ioutil.ReadAll(file); err == nil {
					fileContent = bytes.NewReader(data)

					fs.Cache.Set(filePath, fs.Settings.CacheTTL, &cache.Item{
						ModifiedTime: stat.ModTime(),
						Content:      fileContent,
					})
				}
			}

			if fileContent == nil {
				fileContent = file
			}

			http.ServeContent(w, r, filepath.Base(filePath), stat.ModTime(), fileContent)

			return
		} else {
			fs.handleError(http.StatusInternalServerError, w, r)

			return
		}
	}

	fs.handleError(http.StatusNotFound, w, r)
}

//import (
//	"encoding/json"
//	"fmt"
//	"io"
//	"net/http"
//	"os"
//	"path"
//	"path/filepath"
//	"strings"
//	"time"
//)
//
//// this HTML content will be used as fallback content for 404 response.
//const fallback404content = "<html><body><h1>ERROR 404</h1><h2>Requested file was not found</h2></body></html>"
//
//type json404error struct {
//	Code    int    `json:"code"`
//	Message string `json:"message"`
//}
//
//type (
//	// FileNotFoundHandler handle requests, when requested file was not found on server
//	FileNotFoundHandler func(*FileServer, http.ResponseWriter, *http.Request) (makeStop bool)
//
//	Settings struct {
//		Root         http.Dir
//		IndexFile    string
//		Error404file string
//	}
//
//	FileServer struct {
//		Settings         Settings
//		NotFoundHandlers []FileNotFoundHandler // optional
//	}
//)
//
//// NewFileServer creates new file server.
//func NewFileServer(settings Settings) *FileServer {
//	return &FileServer{
//		Settings: settings,
//		NotFoundHandlers: []FileNotFoundHandler{
//			JSON404errorHandler(),
//			StaticHtmlPage404errorHandler(),
//		},
//	}
//}
//
//// Serve requests to the "public" files and directories.
//func (fileServer *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	// redirect .../index.html to .../
//	if strings.HasSuffix(r.URL.Path, "/"+fileServer.Settings.IndexFile) {
//		http.Redirect(w, r, r.URL.Path[0:len(r.URL.Path)-len(fileServer.Settings.IndexFile)], http.StatusMovedPermanently)
//		return
//	}
//
//	// if empty, set current directory
//	dir := string(fileServer.Settings.Root)
//	if dir == "" {
//		dir = "."
//	}
//
//	// add prefix and clean
//	upath := r.URL.Path
//	if !strings.HasPrefix(upath, "/") {
//		upath = "/" + upath
//		r.URL.Path = upath
//	}
//	// add index file name if requested directory (or server root)
//	if upath[len(upath)-1] == '/' {
//		upath += fileServer.Settings.IndexFile
//	}
//	// make path clean
//	upath = path.Clean(upath)
//
//	// path to file
//	name := path.Join(dir, filepath.FromSlash(upath))
//
//	// if files server root directory is set - try to find file and serve them
//	if len(fileServer.Settings.Root) > 0 {
//		// check for file exists
//		if f, err := os.Open(name); err == nil {
//			// file exists and opened
//			defer func() {
//				if err := f.Close(); err != nil {
//					panic(err)
//				}
//			}()
//			// file (or directory) exists
//			if stat, statErr := os.Stat(name); statErr == nil && stat.Mode().IsRegular() {
//				// requested file is file (not directory)
//				var modTime time.Time
//				// Try to extract file modified time
//				if info, err := f.Stat(); err == nil {
//					modTime = info.ModTime()
//				} else {
//					modTime = time.Now() // fallback
//				}
//				// serve fie content
//				http.ServeContent(w, r, filepath.Base(upath), modTime, f)
//
//				return
//			}
//		}
//	}
//
//	fileServer.handle404(w, r)
//}
//
//func (fileServer *FileServer) handle404(w http.ResponseWriter, r *http.Request) {
//	// If all tries for content serving above has been failed - file was not found (HTTP 404)
//	if fileServer.NotFoundHandlers != nil || len(fileServer.NotFoundHandlers) == 0 {
//		for _, handler := range fileServer.NotFoundHandlers {
//			if handler(fileServer, w, r) {
//				return
//			}
//		}
//	}
//
//	// fallback
//	w.Header().Set("Content-Type", "text/html; charset=utf-8")
//	w.WriteHeader(http.StatusNotFound)
//
//	_, _ = fmt.Fprint(w, fallback404content)
//}
//
//func JSON404errorHandler() FileNotFoundHandler {
//	return func(fs *FileServer, w http.ResponseWriter, r *http.Request) bool {
//		if strings.Contains(r.Header.Get("Accept"), "json") {
//			w.Header().Set("Content-Type", "application/json; charset=utf-8")
//			w.WriteHeader(http.StatusNotFound)
//
//			_ = json.NewEncoder(w).Encode(json404error{
//				Code:    http.StatusNotFound,
//				Message: "Not found",
//			})
//
//			return true
//		}
//
//		return false
//	}
//}
//
//func StaticHtmlPage404errorHandler() FileNotFoundHandler {
//	return func(fs *FileServer, w http.ResponseWriter, r *http.Request) bool {
//		if len(fs.Settings.Root) > 0 {
//			var errPage = path.Join(string(fs.Settings.Root), fs.Settings.Error404file)
//			if f, err := os.Open(errPage); err == nil {
//				// file exists and opened
//				defer func() {
//					if err := f.Close(); err != nil {
//						panic(err)
//					}
//				}()
//
//				// file (or directory) exists
//				if stat, statErr := os.Stat(errPage); statErr == nil && stat.Mode().IsRegular() {
//					w.Header().Set("Content-Type", "text/html; charset=utf-8")
//					w.WriteHeader(http.StatusNotFound)
//
//					// requested file is file (not directory)
//					if _, writeErr := io.Copy(w, f); writeErr != nil {
//						panic(writeErr)
//					}
//
//					return true
//				}
//			}
//		}
//
//		return false
//	}
//}
