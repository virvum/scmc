**WARNING: this software is work in progress.**

[![GoDoc](https://godoc.org/github.com/virvum/scmc?status.svg)](https://godoc.org/github.com/virvum/scmc)
[![Go Report Card](https://goreportcard.com/badge/github.com/virvum/scmc)](https://goreportcard.com/report/github.com/virvum/scmc)

# scmc - Swisscom myCloud utilities and library

This repository contains utilities and a library which interact with
[Swisscom][swisscom]'s [myCloud][mycloud] service (`scmc` stands for
**S**wiss**c**om **m**y**C**loud).

The main motivation behind creating `scmc` is the built-in REST API server
which implements [restic][restic]'s [REST API specification][restic-api] in
order to store restic backups on myCloud.

See `scmc -h` for a full command reference.

[restic]: https://restic.net/
[restic-api]: https://restic.readthedocs.io/en/stable/100_references.html#rest-backend
[swisscom]: https://www.swisscom.ch/
[mycloud]: https://start.mycloud.ch/

## Installing the `scmc` binary

In order to install `scmc` in your `$GOPATH/bin` directory, run the following
command (make sure, that `$GOPATH` is set first):

```sh
go get -v  https://github.com/virvum/scmc/...
```

You can then run `$GOPATH/bin/scmc`. Add `$GOPATH/bin` to your `$PATH` in order
to invoke `scmc` directly from the shell.

## Library

See [examples](examples) for library usage examples.

## Issues and contributions

Please [report any issues found][new-issue] and feel free to [open any pull
requests][new-pr]. Please run `go fmt` and `go vet` against your changes before
opening a pull request.

This project tries to adhere to the following specifications:

* [_Standard_ Go Project Layout][go-layout] for the project layout.
* [Semantic Versioning][semver] for git tags.

[go-layout]: https://github.com/golang-standards/project-layout
[semver]: https://semver.org/
[new-issue]: https://github.com/virvum/scmc/issues/new
[new-pr]: https://github.com/virvum/scmc/compare
