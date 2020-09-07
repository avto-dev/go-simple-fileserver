# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog][keepachangelog] and this project adheres to [Semantic Versioning][semver].

## v1.0.0

### Added

- In memory caching with TTL and limits (like maximal cached files count and maximal file size)
- Overridable error handlers
- "Index" file serving (like `index` [nginx directive](http://nginx.org/en/docs/http/ngx_http_index_module.html#index))
- Redirection to the "parent" directory, when index file requested
- "Allowed methods" list

[keepachangelog]:https://keepachangelog.com/en/1.0.0/
[semver]:https://semver.org/spec/v2.0.0.html
