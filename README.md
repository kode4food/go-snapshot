# go-snapshot
Simple asset file generator for Golang. No recursion, no file system
interface, no restore. Just generates a big map from a compressed byte
array based on the globs you provide.

## How To Install
Make sure your `GOPATH` is set, then run `go get` to retrieve the
package.

```bash
go get gitlab.com/kode4food/go-snapshot
```

## How To Use
Once you've installed the package, you can run it from `GOPATH/bin`
like so:

```bash
go-snapshot -pkg assets -out ./assets/snapshot.go dir1/*.md dir2/*.lisp
```

The generated file exposes a few functions for accessing your assets:

```
// AssetNames returns a sorted string array of the stored asset names
a := assets.AssetNames()

// Get returns an asset as a byte array, and a bool found flag
b, ok := assets.Get("dir1/some_file.md")

// MustGet returns an asset as a byte array, or panics if not found
b := assets.MustGet("dir1/some_file.md")
```
