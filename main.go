package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/niusmallnan/logging-helper/helper"
	"github.com/niusmallnan/logging-helper/resourcewatchers"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/urfave/cli"
)

var VERSION = "v0.1.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "logging-helper"
	app.Version = VERSION
	app.Usage = "A logging helper for Rancher"
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "Debug logging",
			EnvVar: "DEBUG",
		},
		cli.StringFlag{
			Name:  "metadata-address",
			Usage: "The metadata service address",
			Value: "rancher-metadata.rancher.internal",
		},
		cli.IntFlag{
			Name:  "health-check-port",
			Usage: "Port to listen on for healthchecks",
			Value: 9898,
		},
		cli.StringFlag{
			Name:   "docker-graph-dir",
			Usage:  "Root of the Docker runtime",
			Value:  "/var/lib/docker",
			EnvVar: "DOCKER_GRAPH_DIR",
		},
		cli.StringFlag{
			Name:   "logging-containers-dir",
			Usage:  "Root of the docker stdout logging files",
			Value:  "/var/log/logging-containers",
			EnvVar: "LOG_CNT_DIR",
		},
		cli.StringFlag{
			Name:   "logging-volumes-dir",
			Usage:  "Root of the custom logging files ",
			Value:  "/var/log/logging-volumes",
			EnvVar: "LOG_VOL_DIR",
		},
		cli.StringFlag{
			Name:   "logging-volumes-pattern",
			Usage:  "Pattern of the custom logging volumes",
			Value:  "rancher-logging*",
			EnvVar: "LOG_VOL_PATTERN",
		},
		cli.StringFlag{
			Name:   "logging-files-pattern",
			Usage:  "Pattern of the custom logging files",
			Value:  "*.log",
			EnvVar: "LOG_FILE_PATTERN",
		},
	}

	app.Run(os.Args)
}

func run(c *cli.Context) error {
	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	mdClient := metadata.NewClient(fmt.Sprintf("http://%s/2016-07-29", c.String("metadata-address")))
	helper := helper.NewHelper(c)

	exit := make(chan error)

	go func(exit chan<- error) {
		err := resourcewatchers.WatchMetadata(mdClient, helper)
		exit <- errors.Wrap(err, "Metadata watcher exited")

	}(exit)

	go func(exit chan<- error) {
		err := startHealthCheck(c.Int("health-check-port"))
		exit <- errors.Wrapf(err, "Healthcheck provider died.")

	}(exit)

	err := <-exit
	logrus.Errorf("Exiting logging-helper with error: %v", err)
	return err

}

func startHealthCheck(listen int) error {
	http.HandleFunc("/healthcheck", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	})
	logrus.Infof("Listening for health checks on 0.0.0.0:%d/healthcheck", listen)
	err := http.ListenAndServe(fmt.Sprintf(":%d", listen), nil)
	return err
}
