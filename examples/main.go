package main

import (
	"log"
	"net/http"
	"time"

	fileserver "github.com/avto-dev/go-simple-fileserver"
)

func main() {
	fs, err := fileserver.NewFileServer(fileserver.Settings{
		FilesRoot:               "./web",
		IndexFileName:           "index.html",
		ErrorFileName:           "__error__.html",
		RedirectIndexFileToRoot: true,
		AllowedHTTPMethods:      []string{http.MethodGet},
		CacheEnabled:            true,
		CacheTTL:                time.Second * 5,
		CacheMaxFileSize:        1024 * 64, // 64 KiB
		CacheMaxItems:           512,
	})

	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(http.ListenAndServe(":9000", fs))
}
