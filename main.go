package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type fileinfo struct {
	name string
	data []byte
}

type (
	filedata []*fileinfo
	slice    []int
	slices   map[string]slice
)

type config struct {
	pkg  string
	out  string
	in   []string
	data filedata
}

func main() {
	defer handlePanic()
	c := &config{
		out: "assets.go",
		pkg: "main",
	}
	c.run()
}

func handlePanic() {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(-1)
	}
}

func (d filedata) Len() int {
	return len(d)
}

func (d filedata) Less(i, j int) bool {
	return strings.Compare(d[i].name, d[j].name) < 0
}

func (d filedata) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (c *config) run() {
	c.parseConfig()
	c.validateConfig()
	c.read()
	c.validateInput()
	c.write()
}

func (c *config) parseConfig() {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <file patterns>\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.StringVar(&c.pkg, "pkg", c.pkg, "Package name for generated code.")
	flag.StringVar(&c.out, "out", c.out, "Output file to be generated.")
	flag.Parse()

	c.in = make([]string, flag.NArg())
	for i := range c.in {
		c.in[i] = flag.Arg(i)
	}
}

func (c *config) validateConfig() {
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Missing <file pattern>\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

func (c *config) validateInput() {
	if len(c.data) == 0 {
		fmt.Fprintf(os.Stderr, "No assets to bundle\n\n")
		flag.Usage()
		os.Exit(3)
	}
}

func (c *config) read() {
	d := filedata{}
	for _, pattern := range c.in {
		d = append(d, readPattern(pattern)...)
	}
	d.sortFiles()
	c.data = d
}

func (c *config) write() {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(header, c.pkg))
	buf.WriteString(fmt.Sprintf("\n\n%s", c.data.String()))

	if err := os.MkdirAll(path.Dir(c.out), os.ModePerm); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(c.out, buf.Bytes(), os.ModePerm); err != nil {
		panic(err)
	}
}

func (d filedata) sortFiles() {
	sort.Sort(d)
}

func (d filedata) String() string {
	var buf bytes.Buffer
	buf.WriteString(d.decompressor())
	return buf.String()
}

func (d filedata) decompressor() string {
	var buf bytes.Buffer

	sl := len(d)
	buf.WriteString(fmt.Sprintf("var data = make(map[string][]byte, %d)\n", sl))
	buf.WriteString(`
func init() {
`)

	for _, f := range d {
		c := compress(f.data)
		e := fmt.Sprintf("data[%q] = decompress(\"%s\")", f.name, c)
		buf.WriteString(fmt.Sprintf("\t%s\n", e))
	}
	buf.WriteString("}\n")
	return buf.String()
}

func readPattern(pattern string) filedata {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(fmt.Errorf("couldn't resolve pattern %s", pattern))
	}
	d := filedata{}
	for _, filename := range matches {
		d = append(d, readFile(filename))
	}
	return d
}

func readFile(filename string) *fileinfo {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(fmt.Errorf("couldn't read from %s", filename))
	}
	return &fileinfo{
		name: filename,
		data: b,
	}
}

func compress(b []byte) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(b)
	w.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

// here because it's too ugly to go anywhere else
const header = `package %s

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io"
	"sort"
)

// AssetNames returns a list of all assets
func AssetNames() []string {
	an := make([]string, len(data))
	i := 0
	for k := range data {
		an[i] = k
		i++
	}
	sort.Strings(an)
	return an
}

// Get returns an asset by name
func Get(an string) ([]byte, bool) {
	if d, ok := data[an]; ok {
		return d, true
	}
	return nil, false
}

// MustGet returns an asset by name or explodes
func MustGet(an string) []byte {
	if r, ok := Get(an); ok {
		return r
	}
	panic(errors.New("could not find asset: " + an))
}

func decompress(s string) []byte {
	b, _ := base64.StdEncoding.DecodeString(s)
	r, err := gzip.NewReader(bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.Bytes()
}`
