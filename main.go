package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type config struct {
	pkg string
	out string
	in  []string
}

type filedata map[string]string

func main() {
	defer handlePanic()
	c := parseConfig()
	validateConfig(c)
	d := read(c)
	validateInput(d)
	write(c, d)
}

func handlePanic() {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(-1)
	}
}

func parseConfig() *config {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [options] <file patterns>\n\n", os.Args[0])
		flag.PrintDefaults()
	}

	c := &config{
		out: "assets.go",
		pkg: "main",
	}

	flag.StringVar(&c.pkg, "pkg", c.pkg, "Package name for generated code.")
	flag.StringVar(&c.out, "out", c.out, "Output file to be generated.")
	flag.Parse()

	c.in = make([]string, flag.NArg())
	for i := range c.in {
		c.in[i] = flag.Arg(i)
	}

	return c
}

func validateConfig(c *config) {
	if flag.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Missing <file pattern>\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

func validateInput(d filedata) {
	if len(d) == 0 {
		fmt.Fprintf(os.Stderr, "No assets to bundle\n\n")
		flag.Usage()
		os.Exit(3)
	}
}

func read(c *config) filedata {
	d := filedata{}
	for _, pattern := range c.in {
		readPattern(d, pattern)
	}
	return d
}

func readPattern(d filedata, pattern string) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		panic(fmt.Errorf("couldn't resolve pattern %s", pattern))
	}
	for _, filename := range matches {
		readFile(d, filename)
	}
}

func readFile(d filedata, filename string) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(fmt.Errorf("couldn't read from %s", filename))
	}
	d[filename] = compress(b)
}

func compress(b []byte) string {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(b)
	w.Close()

	s := fmt.Sprintf("%x", buf.Bytes())
	p := make([]string, len(s)/2)
	for i, j := 0, 0; i < len(s); i += 2 {
		p[j] = s[i : i+2]
		j++
	}
	return `\x` + strings.Join(p, `\x`)
}

func write(c *config, d filedata) {
	var buf bytes.Buffer
	buf.WriteString(header(c))
	buf.WriteString(filenames(d))
	buf.WriteString(data(d))
	buf.WriteString(functions())

	if err := os.MkdirAll(path.Dir(c.out), os.ModePerm); err != nil {
		panic(err)
	}

	if err := ioutil.WriteFile(c.out, buf.Bytes(), os.ModePerm); err != nil {
		panic(err)
	}
}

func header(c *config) string {
	return fmt.Sprintf(`package %s

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)
`, c.pkg) + "\n"
}

func filenames(d filedata) string {
	var buf bytes.Buffer
	buf.WriteString("var filenames = []string{\n")
	for _, f := range sortedFilenames(d) {
		buf.WriteString(fmt.Sprintf("\t%q,\n", f))
	}
	buf.WriteString("}\n\n")
	return buf.String()
}

func sortedFilenames(d filedata) []string {
	fn := make([]string, len(d))
	i := 0
	for f := range d {
		fn[i] = f
		i++
	}
	sort.Strings(fn)
	return fn
}

func data(d filedata) string {
	var buf bytes.Buffer
	buf.WriteString("var data = map[string][]byte{\n")
	for fn, data := range d {
		buf.WriteString(fmt.Sprintf("\t%q: []byte(\"%s\"),\n", fn, data))
	}
	buf.WriteString("}\n")
	return buf.String()
}

func functions() string {
	return `
// AssetNames returns a sorted list of all bundled paths
func AssetNames() []string {
	an := make([]string, len(filenames))
	for i, n := range filenames {
		an[i] = n
	}
	return an
}

// Get returns an asset by name
func Get(fn string) ([]byte, bool) {
	if d, ok := data[fn]; ok {
		return uncompress(d), true
	}
	return nil, false
}

// MustGet returns an asset by name or explodes
func MustGet(fn string) []byte {
	if r, ok := Get(fn); ok {
		return r
	}
	panic(fmt.Errorf("could not find asset: %s", fn))
}

func uncompress(b []byte) []byte {
	r, err := gzip.NewReader(bytes.NewBuffer(b))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.Bytes()
}`
}
