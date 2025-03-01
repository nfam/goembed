package goembed

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
)

// Generate reads all files in src directory and generates a .go file to dst.
func Generate(src string, dst string, gobuild string, genby string, pkg string, unixTime int64) error {
	root, blob, err := read(src)
	if err != nil {
		return err
	}
	return write(dst, gobuild, genby, pkg, unixTime, root, blob)
}

type file struct {
	path    string
	size    int64
	mime    string
	tag     string
	gz      bool
	gzStart int
	gzEnd   int
	br      bool
	brStart int
	brEnd   int
}

func (f *file) write(w *bytes.Buffer, ind int) {
	w.WriteByte('{')

	// name
	{
		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("name:   ")
		w.WriteByte('"')
		w.WriteString(path.Base(f.path))
		w.WriteByte('"')
		w.WriteByte(',')
	}

	// size
	{
		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("size:   ")
		w.WriteString(strconv.FormatInt(f.size, 10))
		w.WriteByte(',')
	}

	// mime
	{
		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("mime:   ")
		w.WriteByte('"')
		w.WriteString(f.mime)
		w.WriteByte('"')
		w.WriteByte(',')
	}

	// tag
	{
		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("tag:    ")
		w.WriteByte('"')
		w.WriteString(f.tag)
		w.WriteByte('"')
		w.WriteByte(',')
	}

	// gzip
	{
		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("gz:     ")
		if f.gz {
			w.WriteString("true")
		} else {
			w.WriteString("false")
		}
		w.WriteByte(',')

		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("gzBlob: blob[")
		w.WriteString(strconv.Itoa(f.gzStart))
		w.WriteByte(':')
		w.WriteString(strconv.Itoa(f.gzEnd))
		w.WriteByte(']')
		w.WriteByte(',')
	}

	// brotli
	if f.br {
		w.WriteByte('\n')
		indent(w, ind+1)
		w.WriteString("br:     true,\n")
		indent(w, ind+1)
		w.WriteString("brBlob: blob[")
		w.WriteString(strconv.Itoa(f.brStart))
		w.WriteByte(':')
		w.WriteString(strconv.Itoa(f.brEnd))
		w.WriteByte(']')
		w.WriteByte(',')
	}

	w.WriteByte('\n')
	indent(w, ind)
	w.WriteByte('}')
}

func read(dir string) ([]file, []byte, error) {
	var (
		assets []file
		blob   = bytes.NewBuffer(nil)
	)

	var filenames []string
	if err := filepath.Walk(dir, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filenames = append(filenames, filename)
		}
		return nil
	}); err != nil {
		return nil, nil, err
	}
	sort.Strings(filenames)

	for _, filename := range filenames {
		var asset file

		// asset: path
		asset.path, _ = filepath.Rel(dir, filename)
		asset.path = filepath.ToSlash(asset.path)

		// asset: size
		fstat, _ := os.Stat(filename)
		asset.size = fstat.Size()

		// asset: mime
		asset.mime = mime.TypeByExtension(path.Ext(strings.ToLower(asset.path)))
		if asset.mime == "" {
			asset.mime = "application/octet-stream"
		}

		// data
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, nil, err
		}

		// asset: tag
		asset.tag = crc32c(data)

		// compress gzip
		buf := bytes.NewBuffer(nil)
		gz, _ := gzip.NewWriterLevel(buf, gzip.BestCompression)
		if _, err := gz.Write(data); err != nil {
			return nil, nil, err
		}
		if err := gz.Flush(); err != nil {
			return nil, nil, err
		}
		if err := gz.Close(); err != nil {
			return nil, nil, err
		}
		gzData := buf.Bytes()
		asset.gzStart = blob.Len()
		if len(gzData) < len(data) {
			if _, err := blob.Write(gzData); err != nil {
				return nil, nil, err
			}
			asset.gzEnd = asset.gzStart + len(gzData)
			asset.gz = true
		} else {
			if _, err := blob.Write(data); err != nil {
				return nil, nil, err
			}
			asset.gzEnd = asset.gzStart + len(data)
			asset.gz = false
		}

		// compress brotli
		buf = bytes.NewBuffer(nil)
		br := brotli.NewWriterLevel(buf, brotli.BestCompression)
		if _, err := br.Write(data); err != nil {
			return nil, nil, err
		}
		if err := br.Flush(); err != nil {
			return nil, nil, err
		}
		if err := br.Close(); err != nil {
			return nil, nil, err
		}
		brData := buf.Bytes()
		if len(brData) < len(data) && len(brData) < len(gzData) {
			asset.brStart = blob.Len()
			if _, err := blob.Write(brData); err != nil {
				return nil, nil, err
			}
			asset.brEnd = asset.brStart + len(brData)
			asset.br = true
		} else {
			asset.brStart = 0
			asset.brEnd = 0
			asset.br = false
		}

		assets = append(assets, asset)
	}

	return assets, blob.Bytes(), nil
}

//go:embed goembed.go.tpl
var goembed_tpl string

func write(to string, gobuild string, genby string, pkg string, unixTime int64, files []file, blob []byte) error {
	w := bytes.NewBuffer(nil)
	if gobuild != "" {
		w.WriteString("//go:build " + gobuild + "\n\n")
	}
	t := goembed_tpl
	if genby != "" {
		t = strings.Replace(t, "by goembed", "by "+genby, 1)
	}
	if pkg != "" {
		t = strings.Replace(t, "package goembed", "package "+pkg, 1)
	}
	w.WriteString(t)
	if unixTime == 0 {
		unixTime = time.Now().Unix()
	}
	w.WriteString("\nvar stamp = time.Unix(" + strconv.FormatInt(unixTime, 10) + ", 0)\n")

	w.WriteString("\nvar fidx = map[string]int{")
	for i := range files {
		f := &files[i]
		w.WriteString("\n\t\"" + f.path + "\": ")
		w.WriteString(strconv.Itoa(i))
		w.WriteByte(',')
	}
	w.WriteString("\n}\n")

	w.WriteString("\nvar files = [" + strconv.Itoa(len(files)) + "]File{")
	for i := range files {
		f := &files[i]
		w.WriteString("\n\t")
		f.write(w, 1)
		w.WriteByte(',')
	}
	w.WriteString("\n}\n")

	w.WriteString("\n\nvar blob = [...]byte{")
	for i, b := range blob {
		if i%10 == 0 {
			w.WriteByte('\n')
			w.WriteByte('\t')
		} else {
			w.WriteByte(' ')
		}
		w.WriteByte('0')
		w.WriteByte('x')
		w.WriteByte(hex[b>>4])
		w.WriteByte(hex[b&0xF])
		w.WriteByte(',')
	}
	w.WriteString("\n}")

	_ = os.MkdirAll(filepath.Dir(to), os.ModePerm)
	return os.WriteFile(to, w.Bytes(), os.ModePerm)
}
