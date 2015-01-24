package manager

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"github.com/gophergala/docker-bastion/config"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"golang.org/x/crypto/bcrypt"
)

const API_PREFIX = "/api"

func (mgr *Manager) InitMartini() error {
	r := martini.NewRouter()
	m := martini.New()
	m.Use(martini.Recovery())
	m.Use(render.Renderer())
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)

	key, err := mgr.genSessionKey()
	if err != nil {
		log.Error("mgr.genSessionKey: ", err)
		return err
	}

	ssStore := sessions.NewCookieStore(key)
	m.Use(sessions.Sessions("SID", ssStore))
	m.Use(mgr.authMidware)
	mgr.m = &martini.ClassicMartini{m, r}
	mgr.registerRoutes()
	return nil
}

type Admin struct {
	Id       int
	Name     string
	Password string
}

func (mgr *Manager) authMidware(c martini.Context, w http.ResponseWriter, r *http.Request, rnd render.Render, ss sessions.Session) {
	if !mgr.authRequired(r) {
		return
	}

	uid_ := ss.Get("uid")
	if uid_ == nil {
		w.WriteHeader(403)
		return
	}
	uid, ok := uid_.(int)
	if !ok {
		w.WriteHeader(403)
		return
	}

	stmt, err := mgr.db.Prepare("select 1 from admins where id == ?")
	if err == nil {
		exists := 0
		stmt.QueryRow(uid).Scan(&exists)
		if exists == 1 {
			return
		}
	}
	w.WriteHeader(403)
}

func (mgr *Manager) genSessionKey() ([]byte, error) {
	var (
		keyPath = config.DATA_DIR + "/session.key"
		key     []byte
		err     error
	)
	if key, err = ioutil.ReadFile(keyPath); err != nil {
		key = make([]byte, 171)
		_, err = rand.Read(key)
		if err != nil {
			return nil, err
		}
		ioutil.WriteFile(keyPath, key, 0400)
	}
	return key, nil
}

var AuthWhiteList = map[string][]string{
	"GET":    {"/_ping"},
	"POST":   {"/login"},
	"DELETE": {},
}

func (mgr *Manager) authRequired(r *http.Request) bool {
	list, ok := AuthWhiteList[r.Method]
	if !ok {
		return false
	}
	for _, path := range list {
		if r.URL.Path == API_PREFIX+path {
			return false
		}
	}
	return true
}

func (mgr *Manager) registerRoutes() {
	mgr.m.Group(API_PREFIX, func(r martini.Router) {
		r.Get("/_ping", func(w http.ResponseWriter) {
			w.Write([]byte{'O', 'K'})
		})
		r.Post("/login", mgr.Login)
	})
}

func (mgr *Manager) showError(err error, w http.ResponseWriter) {
	status := 500
	switch err {
	case sql.ErrNoRows:
		status = 404
	case bcrypt.ErrMismatchedHashAndPassword:
		status = 400
	}
	w.WriteHeader(status)
}

func (mgr *Manager) Login(w http.ResponseWriter, r *http.Request, rnd render.Render, ss sessions.Session) {
	req := map[string]string{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || len(req["name"]) < 3 || len(req["password"]) < 3 {
		w.WriteHeader(400)
		return
	}

	var (
		password string
		uid      int
	)
	stmt, err := mgr.db.Prepare("select id, password from admins where name = ?")
	if err == nil {
		err = stmt.QueryRow(req["name"]).Scan(&uid, &password)
	}
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(password), []byte(req["password"]))
	}
	if err != nil {
		mgr.showError(err, w)
		return
	}
	ss.Set("uid", uid)
	w.WriteHeader(204)
}
