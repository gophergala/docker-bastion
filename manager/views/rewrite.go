package views

import (
	"net/http"

	"github.com/martini-contrib/sessions"
)

var rewriteMap = map[string]string{
	"/signin":     "/signin.html",
	"/containers": "/dashboard.html",
	"/users":      "/dashboard.html",
}

func (v *Views) Rewrite(w http.ResponseWriter, r *http.Request, ss sessions.Session) bool {
	if r.URL.Path == "/" {
		if ss.Get("uid") == nil {
			r.URL.Path = "/" + v.Index
		} else {
			http.Redirect(w, r, "/containers", 302)
			return true
		}
	}

	if t, ok := rewriteMap[r.URL.Path]; ok {
		r.URL.Path = t
	}
	return false
}
