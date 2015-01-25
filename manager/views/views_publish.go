// +build publish

package views

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/martini-contrib/sessions"
)

type Views struct {
	Names map[string]struct{}
	Index string
}

func New(fallback string) *Views {
	v := &Views{make(map[string]struct{}), fallback}
	names := AssetNames()
	for _, n := range names {
		v.Names[n] = struct{}{}
	}
	return v
}

func (v *Views) ServeHTTP(w http.ResponseWriter, r *http.Request, ss sessions.Session) {
	v.Rewrite(r, ss)
	var ok bool
	name := r.URL.Path[1:]
	if _, ok = v.Names[name]; !ok {
		name = v.Index
	}

	ext := path.Ext(name)
	ct := mime.TypeByExtension(ext)
	w.Header().Set("Content-Type", ct)

	data, _ := Asset(name)
	hdr := r.Header.Get("Accept-Encoding")
	if strings.Contains(hdr, "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		w.Write(data)
	} else {
		gz, err := gzip.NewReader(bytes.NewBuffer(data))
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		io.Copy(w, gz)
		gz.Close()
	}
}
