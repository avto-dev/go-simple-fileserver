# Simple golang file server

![Release version][badge_release_version]
![Project language][badge_language]
[![Build Status][badge_build]][link_build]
[![Coverage][badge_coverage]][link_coverage]
[![Go Report][badge_goreport]][link_goreport]
[![License][badge_license]][link_license]

This package provides basic file server functionality with:

- In memory caching with TTL and limits (like maximal cached files count and maximal file size)
- Overridable error handlers
- "Index" file serving (like `index` [nginx directive](http://nginx.org/en/docs/http/ngx_http_index_module.html#index))
- Redirection to the "parent" directory, when index file requested
- "Allowed methods" list

Most use-case is [SPA](https://en.wikipedia.org/wiki/Single-page_application) assets serving.

## Example

```go
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
```

To run this example execute `go run .` in `./examples` directory.

More information can be found in the godocs: <http://godoc.org/github.com/avto-dev/go-simple-fileserver>

### Testing

For package testing we use built-in golang testing feature and `docker-ce` + `docker-compose` as develop environment. So, just write into your terminal after repository cloning:

```shell
$ make test
```

## Changelog

[![Release date][badge_release_date]][link_releases]
[![Commits since latest release][badge_commits_since_release]][link_commits]

Changes log can be [found here][link_changes_log].

## Support

[![Issues][badge_issues]][link_issues]
[![Issues][badge_pulls]][link_pulls]

If you will find any package errors, please, [make an issue][link_create_issue] in current repository.

## License

This is open-sourced software licensed under the [MIT License][link_license].

[badge_build]:https://img.shields.io/github/workflow/status/avto-dev/go-simple-fileserver/tests/master
[badge_coverage]:https://img.shields.io/codecov/c/github/avto-dev/go-simple-fileserver/master.svg?maxAge=30
[badge_goreport]:https://goreportcard.com/badge/github.com/avto-dev/go-simple-fileserver
[badge_release_version]:https://img.shields.io/github/release/avto-dev/go-simple-fileserver.svg?maxAge=30
[badge_language]:https://img.shields.io/github/go-mod/go-version/avto-dev/go-simple-fileserver?longCache=true
[badge_license]:https://img.shields.io/github/license/avto-dev/go-simple-fileserver.svg?longCache=true
[badge_release_date]:https://img.shields.io/github/release-date/avto-dev/go-simple-fileserver.svg?maxAge=180
[badge_commits_since_release]:https://img.shields.io/github/commits-since/avto-dev/go-simple-fileserver/latest.svg?maxAge=45
[badge_issues]:https://img.shields.io/github/issues/avto-dev/go-simple-fileserver.svg?maxAge=45
[badge_pulls]:https://img.shields.io/github/issues-pr/avto-dev/go-simple-fileserver.svg?maxAge=45
[link_goreport]:https://goreportcard.com/report/github.com/avto-dev/go-simple-fileserver

[link_coverage]:https://codecov.io/gh/avto-dev/go-simple-fileserver
[link_build]:https://github.com/avto-dev/go-simple-fileserver/actions
[link_license]:https://github.com/avto-dev/go-simple-fileserver/blob/master/LICENSE
[link_releases]:https://github.com/avto-dev/go-simple-fileserver/releases
[link_commits]:https://github.com/avto-dev/go-simple-fileserver/commits
[link_changes_log]:https://github.com/avto-dev/go-simple-fileserver/blob/master/CHANGELOG.md
[link_issues]:https://github.com/avto-dev/go-simple-fileserver/issues
[link_create_issue]:https://github.com/avto-dev/go-simple-fileserver/issues/new/choose
[link_pulls]:https://github.com/avto-dev/go-simple-fileserver/pulls
