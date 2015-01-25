package main

import (
	"database/sql"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gophergala/docker-bastion/config"
	"github.com/gophergala/docker-bastion/manager"
	"github.com/gophergala/docker-bastion/sshd"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	app := cli.NewApp()
	app.Name = "docker-bastion"
	app.Usage = "Allow remote accessing docker containers via SSH"
	app.Version = "0.1.0"
	app.Author = "Shijiang Wei"
	app.Email = "mountkin@gmail.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "ssh-addr",
			Usage: "the address that SSH listens",
			Value: ":2222",
		},
		cli.StringFlag{
			Name:  "manage-addr",
			Usage: "the address that the management listens",
			Value: ":1015",
		},
	}

	app.Action = run
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func run(c *cli.Context) {
	errs := make(chan error)
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	ssh, err := sshd.New(c.String("ssh-addr"), db, errs)
	if err != nil {
		log.Fatal(err)
	}
	mgr, err := manager.New(c.String("manage-addr"), db, errs)
	if err != nil {
		log.Fatal(err)
	}

	ssh.Start()
	mgr.Start()
	for i := 0; i < 2; i++ {
		err := <-errs
		if err != nil {
			log.Fatal(err)
		}
	}
}

func initDB() (*sql.DB, error) {
	dbPath := config.DATA_DIR + "/meta.sqlite3"
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	stmts := []string{`create table if not exists users (
			id integer primary key,
			name character(100) unique,
			password character(80),
			created_at datetime default current_timestamp
		)`,
		`create index if not exists idx_name on users (name)`,
		`create table if not exists containers (
			id integer primary key,
			cid character(12),
			user_id integer,
			created_at datetime default current_timestamp
		)`,
		`create unique index if not exists idx_cid_uid on containers (cid, user_id)`,
		`create table if not exists admins (
			id integer primary key,
			name character(100) unique,
			password character(80),
			created_at datetime default current_timestamp
		)`,
		`create index if not exists idx_name on admins (name)`,
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return nil, err
		}
	}
	return db, nil
}
