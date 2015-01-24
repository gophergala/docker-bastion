package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/gophergala/docker-bastion/manager"
	"github.com/gophergala/docker-bastion/sshd"
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
	ssh, err := sshd.New(c.String("ssh-addr"), errs)
	if err != nil {
		log.Fatal(err)
	}
	mgr, err := manager.New(c.String("manage-addr"), errs)
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
