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

// ErrorHandlerFunc is used as handler for errors processing. If func return `true` - next handler will be NOT executed.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, fs *FileServer, errorCode int) (doNotContinue bool)

// FileServer is a main file server structure (implements `http.Handler` interface).
type FileServer struct {
	// Server settings (some of them can be changed in runtime).
	Settings Settings

	// Cacher instance.
	Cache cache.Cacher // nil, if caching disabled

	// If all error handlers fails - this content will be used as fallback for error page generating.
	FallbackErrorContent string

	// Error handlers stack.
	ErrorHandlers []ErrorHandlerFunc

	// Allowed HTTP methods map (is used in performance reasons).
	allowedHTTPMethodsMap map[string]struct{} // fillable in runtime
}

// Settings describes file server options.
type Settings struct {
	// Directory path, where files for serving is located.
	FilesRoot string

	// File name (relative path to the file) that will be used as an index (like <https://bit.ly/356QeFm>).
	IndexFileName string

	// File name (relative path to the file) that will be used as error page template.
	ErrorFileName string

	// Respond "index file" request with redirection to the root (`example.com/index.html` -> `example.com/`).
	RedirectIndexFileToRoot bool

	// Allowed HTTP methods (eg.: `http.MethodGet`).
	AllowedHTTPMethods []string

	// Enables caching engine.
	CacheEnabled bool

	// Maximal data caching lifetime.
	CacheTTL time.Duration

	// Maximum file size (in bytes), that can be placed into the cache.
	CacheMaxFileSize int64

	// Maximum files count, that can be placed into the cache.
	CacheMaxItems uint32
}

// NewFileServer creates new file server with default settings. Feel free to change default behavior.
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

	if len(s.AllowedHTTPMethods) == 0 {
		s.AllowedHTTPMethods = append(s.AllowedHTTPMethods, http.MethodGet)
	}

	fs := &FileServer{
		Settings:             s,
		FallbackErrorContent: defaultFallbackErrorContent,
	}

	if s.CacheEnabled {
		fs.Cache = cache.NewInMemoryCache(s.CacheTTL / 2) //nolint:gomnd
	}

	fs.ErrorHandlers = []ErrorHandlerFunc{
		JSONErrorHandler(),
		StaticHTMLPageErrorHandler(),
	}

	return fs, nil
}

// CacheAvailable checks cache availability.
func (fs *FileServer) CacheAvailable() bool {
	return fs.Settings.CacheEnabled && fs.Cache != nil
}

func (fs *FileServer) handleError(w http.ResponseWriter, r *http.Request, errorCode int) {
	if fs.ErrorHandlers != nil && len(fs.ErrorHandlers) > 0 {
		for _, handler := range fs.ErrorHandlers {
			if handler(w, r, fs, errorCode) {
				return
			}
		}
	}

	// fallback
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(errorCode)

	_, _ = w.Write([]byte(ErrorPageTemplate(fs.FallbackErrorContent).Build(errorCode)))
}

func (fs *FileServer) methodIsAllowed(method string) bool {
	if fs.allowedHTTPMethodsMap == nil {
		// burn allowed methods map for fast checking
		fs.allowedHTTPMethodsMap = make(map[string]struct{})

		for _, v := range fs.Settings.AllowedHTTPMethods {
			fs.allowedHTTPMethodsMap[v] = struct{}{}
		}
	}

	_, found := fs.allowedHTTPMethodsMap[method]

	return found
}

// ServeHTTP responds to an HTTP request.
func (fs *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) { //nolint:funlen,gocognit,gocyclo
	if !fs.methodIsAllowed(r.Method) {
		fs.handleError(w, r, http.StatusMethodNotAllowed)

		return
	}

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
	if len(fs.Settings.IndexFileName) > 0 && urlPath[len(urlPath)-1] == '/' {
		urlPath += fs.Settings.IndexFileName
	}

	// prepare target file path
	filePath := path.Join(fs.Settings.FilesRoot, filepath.FromSlash(path.Clean(urlPath)))

	// look for response in cache
	if fs.CacheAvailable() {
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

			// put file content into cache, if it is possible
			if fs.CacheAvailable() &&
				fs.Cache.Count() < fs.Settings.CacheMaxItems &&
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
		}

		fs.handleError(w, r, http.StatusInternalServerError)

		return
	}

	fs.handleError(w, r, http.StatusNotFound)
}
