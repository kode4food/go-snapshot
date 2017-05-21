# go-snapshot
Simple asset file generator for Golang. No recursion, no file system interface, no restore. Just generates a big map of byte arrays based on the globs you provide.

## How To Install
Make sure your `GOPATH` is set, then run `go get` to retrieve the
package.

```bash
go get github.com/kode4food/go-snapshot
```

## How To Use
Once you've installed the package, you can run it from `GOPATH/bin`
like so:

```bash
go-snapshot ./dir1/*.md ./dir2/*.lisp -out ./assets/snapshot.go
```

