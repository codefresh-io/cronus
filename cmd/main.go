package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/codefresh-io/cronus/pkg/backend"
	"github.com/codefresh-io/cronus/pkg/cron"
	"github.com/codefresh-io/cronus/pkg/cronexp"
	"github.com/codefresh-io/cronus/pkg/hermes"
	"github.com/codefresh-io/cronus/pkg/types"
	"github.com/codefresh-io/cronus/pkg/version"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var runner *cron.Runner
var store types.EventStore
var cronguru cronexp.Service

// HermesDryRun dry run stub
type HermesDryRun struct {
}

// TriggerEvent dry run version
func (m *HermesDryRun) TriggerEvent(eventURI string, event *hermes.NormalizedEvent) error {
	fmt.Println(eventURI)
	fmt.Println("\tSecret: ", event.Secret)
	fmt.Println("\tVariables:")
	for k, v := range event.Variables {
		fmt.Println("\t\t", k, "=", v)
	}
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "cronus"
	app.Authors = []cli.Author{{Name: "Alexei Ledenev", Email: "alexei@codefresh.io"}}
	app.Version = version.HumanVersion
	app.EnableBashCompletion = true
	app.Usage = "CRON event generation"
	app.UsageText = fmt.Sprintf(`Run Cronus CRON Event Provider server.
%s
cronus respects the following environment variables:

   - HERMES_SERVICE     - set the url to the Hermes service (default "hermes")
   
Copyright © Codefresh.io`, version.ASCIILogo)
	app.Before = before

	app.Commands = []cli.Command{
		{
			Name: "server",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "hermes",
					Usage:  "Codefresh Hermes service",
					Value:  "http://hermes/",
					EnvVar: "HERMES_SERVICE",
				},
				cli.StringFlag{
					Name:   "token, t",
					Usage:  "Codefresh Hermes API token",
					Value:  "TOKEN",
					EnvVar: "HERMES_TOKEN",
				},
				cli.IntFlag{
					Name:  "port",
					Usage: "TCP port for the dockerhub provider server",
					Value: 8080,
				},
				cli.BoolFlag{
					Name:  "dry-run",
					Usage: "do not execute triggers, just log to console",
				},
			},
			Usage: "start cronus server",
			Description: `Run Cronus CRON Event Provider server. Cronus generates time-based events and sends normalized event payload to the Codefresh Hermes trigger manager service to invoke associated Codefresh pipelines.
			
		Event URI Pattern: cron:codefresh:{{cron-expression}}:{{message}}`,
			Action: runServer,
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level, l",
			Usage:  "set log level (debug, info, warning(*), error, fatal, panic)",
			Value:  "warning",
			EnvVar: "LOG_LEVEL",
		},
		cli.BoolFlag{
			Name:  "json",
			Usage: "produce log in JSON format: Logstash and Splunk friendly",
		},
	}

	app.Run(os.Args)

}

func before(c *cli.Context) error {
	// set debug log level
	switch level := c.GlobalString("log-level"); level {
	case "debug", "DEBUG":
		log.SetLevel(log.DebugLevel)
	case "info", "INFO":
		log.SetLevel(log.InfoLevel)
	case "warning", "WARNING":
		log.SetLevel(log.WarnLevel)
	case "error", "ERROR":
		log.SetLevel(log.ErrorLevel)
	case "fatal", "FATAL":
		log.SetLevel(log.FatalLevel)
	case "panic", "PANIC":
		log.SetLevel(log.PanicLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}
	// set log formatter to JSON
	if c.GlobalBool("json") {
		log.SetFormatter(&log.JSONFormatter{})
	}

	return nil
}

// start trigger manager server
func runServer(c *cli.Context) error {
	fmt.Println()
	fmt.Println(version.ASCIILogo)

	// setup gin router
	router := gin.New()
	router.Use(gin.Recovery())
	// event info route
	router.GET("/cronus/event/:uri/:secret", gin.Logger(), getEventInfo)
	router.GET("/event/:uri/:secret", gin.Logger(), getEventInfo)
	// subscribe/unsubscribe route
	router.POST("/cronus/event/:uri/:secret", gin.Logger(), subscribeToEvent)
	router.POST("/event/:uri/:secret", gin.Logger(), subscribeToEvent)
	router.DELETE("/cronus/event/:uri", gin.Logger(), unsubscribeFromEvent)
	router.DELETE("/event/:uri", gin.Logger(), unsubscribeFromEvent)
	// status routes
	router.GET("/cronus/health", getHealth)
	router.GET("/health", getHealth)
	router.GET("/cronus/version", getVersion)
	router.GET("/version", getVersion)
	router.GET("/", getVersion)

	// access hermes
	var hermesSvc hermes.Service
	if c.Bool("dry-run") {
		hermesSvc = &HermesDryRun{}
	} else {
		// add http protocol, if missing
		hermesSvcName := c.String("hermes")
		if !strings.HasPrefix(hermesSvcName, "http://") {
			hermesSvcName = "http://" + hermesSvcName
		}
		hermesSvc = hermes.NewHermesEndpoint(hermesSvcName, c.String("token"))
	}
	// access boltdb
	store, err := backend.NewBoltEventStore("/var/events.db")
	if err != nil {
		return err
	}
	// start cron runner
	runner = cron.NewCronRunner(store, hermesSvc)
	// create cronguru service for cron expression description
	cronguru = cronexp.NewCronDescriptorEndpoint()

	// run server
	port := c.Int("port")
	log.WithField("port", port).Debug("starting cronus server")
	return router.Run(fmt.Sprintf(":%d", port))
}

func getEventInfo(c *gin.Context) {
	uri := c.Param("uri")
	// get event
	event, err := store.GetEvent(uri)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, event)
}

func subscribeToEvent(c *gin.Context) {
	uri := c.Param("uri")
	secret := c.Param("secret")
	event, err := types.ConstructEvent(uri, secret, cronguru)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// add cron job
	err = runner.AddCronJob(*event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

func unsubscribeFromEvent(c *gin.Context) {
	uri := c.Param("uri")
	// remove cron job
	err := runner.RemoveCronJob(uri)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func getHealth(c *gin.Context) {
	c.Status(http.StatusOK)
}

func getVersion(c *gin.Context) {
	c.String(http.StatusOK, version.HumanVersion)
}
