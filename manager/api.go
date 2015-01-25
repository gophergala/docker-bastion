package manager

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"github.com/gophergala/docker-bastion/config"
	"github.com/gophergala/docker-bastion/manager/views"
	"github.com/martini-contrib/render"
	"github.com/martini-contrib/sessions"
	"github.com/mountkin/dockerclient"
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

func (mgr *Manager) authMidware(c martini.Context, w http.ResponseWriter, r *http.Request, ss sessions.Session) {
	if !strings.HasPrefix(r.URL.Path, API_PREFIX) {
		return
	}

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

	exists := 0
	mgr.db.QueryRow("select 1 from admins where id == ?", uid).Scan(&exists)
	if exists == 1 {
		return
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
	view := views.New("signin.html")
	mgr.m.NotFound(view.ServeHTTP)
	mgr.m.Group(API_PREFIX, func(r martini.Router) {
		r.Get("/_ping", func(w http.ResponseWriter) {
			w.Write([]byte{'O', 'K'})
		})
		r.Post("/login", mgr.Login)
		r.Delete("/logout", mgr.Logout)
		r.Post("/passwd", mgr.ChPasswd)
		r.Post("/users", mgr.AddUser)
		r.Get("/users", mgr.Users)
		r.Delete("/users/:id", mgr.DeleteUser)
		r.Post("/priv", mgr.Grant)
		r.Delete("/priv/:id", mgr.Revoke)
		r.Post("/containers", mgr.CreateContainer)
		r.Delete("/containers/:id", mgr.DeleteContainer)
		r.Get("/containers", mgr.Containers)
	})
}

func (mgr *Manager) showError(err error, w http.ResponseWriter) {
	status := 500
	switch err.(type) {
	case *strconv.NumError:
		status = 400
	default:
		switch err {
		case sql.ErrNoRows:
			status = 404
		case bcrypt.ErrMismatchedHashAndPassword:
			status = 400
		}
	}
	w.WriteHeader(status)
	fmt.Fprintf(w, "{%q:%q}", "message", err.Error())
}

// POST /api/login
func (mgr *Manager) Login(w http.ResponseWriter, r *http.Request, ss sessions.Session) {
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
	err = mgr.db.QueryRow("select id, password from admins where name = ?", req["name"]).Scan(&uid, &password)
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

// DELETE /api/logout
func (mgr *Manager) Logout(w http.ResponseWriter, r *http.Request, ss sessions.Session) {
	ss.Delete("uid")
	w.WriteHeader(204)
}

type User struct {
	Id        int       `json:"id,omitempty"`
	Name      string    `json:"user_name"`
	CreatedAt time.Time `json:"user_created_at"`
}

// User to Container privileges map
type Priv struct {
	PrivId   int       `json:"priv_id"`
	Cid      string    `json:"cid"`
	UserId   int       `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}

type UserPriv struct {
	User
	Priv
}

// POST /api/users
func (mgr *Manager) AddUser(w http.ResponseWriter, r *http.Request, rnd render.Render) {
	req := make(map[string]string)
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || len(req["name"]) < 3 || len(req["password"]) < 3 {
		w.WriteHeader(400)
		return
	}
	passwd, err := bcrypt.GenerateFromPassword([]byte(req["password"]), 11)
	if err != nil {
		log.Error("bcrypt.GenerateFromPassword: ", err)
		w.WriteHeader(500)
		return
	}

	var (
		result sql.Result
		id     int64
	)
	result, err = mgr.db.Exec("insert into users (name, password) values(?,?)", req["name"], passwd)
	if err == nil {
		id, err = result.LastInsertId()
	}
	if err != nil {
		mgr.showError(err, w)
		return
	}
	rnd.JSON(200, map[string]interface{}{"name": req["name"], "id": id})
}

// GET /api/users
func (mgr *Manager) Users(w http.ResponseWriter, rnd render.Render) {
	ret := []UserPriv{}
	rows, err := mgr.db.Query("select users.id, users.name, users.created_at, containers.id priv_id, containers.cid, containers.created_at joined_at from users left join containers on users.id = containers.user_id order by users.id, containers.id")
	if err != nil {
		mgr.showError(err, w)
		return
	}
	defer rows.Close()

	for rows.Next() {
		user := UserPriv{}
		err = rows.Scan(&user.UserId, &user.Name, &user.CreatedAt, &user.PrivId, &user.Cid, &user.JoinedAt)
		ret = append(ret, user)
	}
	rnd.JSON(200, ret)
}

// DELETE /api/users/:id
func (mgr *Manager) DeleteUser(w http.ResponseWriter, params martini.Params) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		mgr.showError(err, w)
		return
	}
	tx, err := mgr.db.Begin()
	if err != nil {
		mgr.showError(err, w)
		return
	}

	_, err = tx.Exec("delete from users where id = ?", id)
	if err == nil {
		_, err = tx.Exec("delete from containers where user_id = ?", id)
	}
	if err == nil {
		err = tx.Commit()
	} else {
		tx.Rollback()
	}
	if err != nil {
		mgr.showError(err, w)
	}
	w.WriteHeader(204)
}

// POST /api/priv?user_id=123&container=dev1
func (mgr *Manager) Grant(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	if len(r.Form["user_id"]) != 1 || len(r.Form["container"]) != 1 {
		w.WriteHeader(400)
		fmt.Fprintf(w, "{%q:%q}", "message", "user_id and container are required")
		return
	}
	uid, err := strconv.Atoi(r.Form["user_id"][0])
	if err != nil {
		mgr.showError(err, w)
		return
	}

	// TODO: check existence of the user
	container := r.Form["container"][0]
	result, err := mgr.db.Exec("insert into containers (user_id, cid) values (?, ?)", uid, container)
	if err != nil {
		mgr.showError(err, w)
		return
	}
	id, _ := result.LastInsertId()
	json.NewEncoder(w).Encode(map[string]int64{"id": id})
}

// DELETE /api/priv/:id
func (mgr *Manager) Revoke(w http.ResponseWriter, params martini.Params) {
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		mgr.showError(err, w)
		return
	}
	_, err = mgr.db.Exec("delete from containers where id = ?", id)
	if err != nil {
		mgr.showError(err, w)
	} else {
		w.WriteHeader(204)
	}
}

// POST /api/containers
func (mgr *Manager) CreateContainer(w http.ResponseWriter, r *http.Request) {
	params := make(map[string]string)
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil || len(params["name"]) < 3 || len(params["image"]) < 3 {
		w.WriteHeader(400)
		return
	}

	cfg := &dockerclient.ContainerConfig{
		Tty:   true,
		Image: params["image"],
		Cmd:   []string{"/bin/sh"},
		HostConfig: dockerclient.HostConfig{
			RestartPolicy: dockerclient.RestartPolicy{"always", 125},
		},
	}
	cid, err := mgr.client.CreateContainer(cfg, params["name"])
	if err == dockerclient.ErrNotFound {
		err = mgr.client.PullImage(cfg.Image, nil)
		if err == nil {
			cid, err = mgr.client.CreateContainer(cfg, params["name"])
		}
	}
	if err != nil {
		log.Error("mgr.client.CreateContainer: ", err)
		mgr.showError(err, w)
	} else {
		_, err = mgr.db.Exec("insert into containers (cid, user_id) values (?, ?)", cid, params["user_id"])
		if err != nil {
			mgr.showError(err, w)
			return
		}
		mgr.client.StartContainer(cid, nil)
		fmt.Fprintf(w, "{%q:%q}", "cid", cid)
	}
}

// DELETE /api/containers/:id
func (mgr *Manager) DeleteContainer(w http.ResponseWriter, r *http.Request, params martini.Params, rnd render.Render) {
	err := mgr.client.RemoveContainer(params["id"], true)
	if err == nil {
		mgr.db.Exec("delete from containers where cid = ?", params["id"])
		w.WriteHeader(204)
	} else {
		log.Error(err)
		rnd.JSON(500, map[string]string{"message": err.Error()})
	}
}

// GET /api/containers
func (mgr *Manager) Containers(w http.ResponseWriter, rnd render.Render) {
	containers, err := mgr.client.ListContainers(true, false, "")
	if err != nil {
		log.Error(err)
		w.WriteHeader(500)
		return
	}
	rnd.JSON(200, containers)
}

// GET /api/containers
func (mgr *Manager) ChPasswd(w http.ResponseWriter, r *http.Request, ss sessions.Session) {
	params := make(map[string]string)
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil || len(params["password"]) < 5 {
		w.WriteHeader(400)
		return
	}

	passwd, err := bcrypt.GenerateFromPassword([]byte(params["password"]), 11)
	if err != nil {
		mgr.showError(err, w)
		return
	}
	uid := ss.Get("uid").(int)
	_, err = mgr.db.Exec("update admins set password = ? where id = ?", passwd, uid)
	if err != nil {
		log.Error("ChPasswd: ", err)
		mgr.showError(err, w)
		return
	}
	w.WriteHeader(204)
}
