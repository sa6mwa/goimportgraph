# goimportgraph

Goimportgraph lists the repositories behind `go list -mod=readonly -m all` by
crawling for the `go-import` meta tags.

```console
$ go install github.com/sa6mwa/goimportgraph@latest
```

Or use the `Makefile`...

```console
$ make
go build -o goimportgraph -ldflags=-s .

$ sudo make install
install goimportgraph /usr/local/bin/goimportgraph
```
