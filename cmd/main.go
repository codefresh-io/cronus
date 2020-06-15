package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/codefresh-io/cronus/pkg/backend"
	"github.com/codefresh-io/cronus/pkg/cron"
	"github.com/codefresh-io/cronus/pkg/cronexp"
	"github.com/codefresh-io/cronus/pkg/hermes"
	"github.com/codefresh-io/cronus/pkg/types"
	"github.com/codefresh-io/cronus/pkg/version"
	"github.com/codefresh-io/go-infra/pkg/logger"
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
					Value:  "http://local.codefresh.io:9011",
					EnvVar: "HERMES_SERVICE",
				},
				cli.StringFlag{
					Name:   "token, t",
					Usage:  "Codefresh Hermes API token",
					Value:  "TOKEN",
					EnvVar: "HERMES_TOKEN",
				},
				cli.StringFlag{
					Name:   "store",
					Usage:  "BoltDB storage file",
					Value:  "/var/tmp/events.db",
					EnvVar: "STORE_FILE",
				},
				cli.IntFlag{
					Name:   "port",
					Usage:  "TCP port for the cronus provider server",
					EnvVar: "PORT",
					Value:  10002,
				},
				cli.Int64Flag{
					Name:   "limit",
					Usage:  "minimal allowed cron interval (seconds)",
					EnvVar: "LIMIT",
					Value:  60,
				},
				cli.BoolFlag{
					Name:  "dry-run",
					Usage: "do not execute triggers, just log to console",
				},
			},
			Usage: "start cronus server",
			Description: `Run Cronus CRON Event Provider server. Cronus generates time-based events and sends normalized event payload to the Codefresh Hermes trigger manager service to invoke associated Codefresh pipelines.
			
		Event URI Pattern: cron:codefresh:{{cron-expression}}:{{message}}[:{{account}}]`,
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
			Name:   "json",
			Usage:  "produce log in Codefresh JSON format",
			EnvVar: "LOG_JSON",
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
		log.SetFormatter(&logger.CFFormatter{})
	}
	// trace function calls
	traceHook := logger.NewHook()
	traceHook.Prefix = "codefresh:hermes:"
	traceHook.AppName = "hermes"
	traceHook.FunctionField = logger.FieldNamespace
	traceHook.AppField = logger.FieldService
	log.AddHook(traceHook)

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
	router.POST("/cronus/event/:uri/:secret/*creds", gin.Logger(), subscribeToEvent)
	router.POST("/event/:uri/:secret/*creds", gin.Logger(), subscribeToEvent)
	router.DELETE("/cronus/event/:uri/*creds", gin.Logger(), unsubscribeFromEvent)
	router.DELETE("/event/:uri/*creds", gin.Logger(), unsubscribeFromEvent)
	// status routes
	router.GET("/cronus/health", getHealth)
	router.GET("/health", getHealth)
	router.GET("/cronus/version", getVersion)
	router.GET("/version", getVersion)
	router.GET("/cronus/ping", ping)
	router.GET("/ping", ping)
	router.GET("/backup", backupDB)
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
	var err error
	log.WithField("store", store).Debug("initializing BoltDB")
	store, err = backend.NewBoltEventStore(c.String("store"))
	if err != nil {
		log.WithError(err).Error("failed to start BoltDB")
		return err
	}
	// start cron runner
	log.Debug("starting cron job runner")
	runner = cron.NewCronRunner(store, hermesSvc, c.Int64("limit"))
	// create cronguru service for cron expression description
	cronguru = cronexp.NewCronExpression()

	// set server port
	port := c.Int("port")
	log.WithField("port", port).Debug("starting cronus server")
	// use RawPath: the url.RawPath will be used to find parameters
	router.UseRawPath = true
	// run server
	return router.Run(fmt.Sprintf(":%d", port))
}

func getParam(c *gin.Context, name string) string {
	v := c.Param(name)
	v, err := url.PathUnescape(v)
	if err != nil {
		log.WithFields(log.Fields{
			"name":  name,
			"value": v,
		}).WithError(err).Error("failed to URL decode value")
	}
	return v
}

func getEventInfo(c *gin.Context) {
	uri := getParam(c, "uri")
	log.WithField("uri", c.Param("uri")).Debug("get event details")
	// get event
	event, err := store.GetEvent(uri)
	if err != nil {
		log.WithError(err).Error("failed to get event info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, event)
}

func subscribeToEvent(c *gin.Context) {
	uri := getParam(c, "uri")
	log.WithField("uri", uri).Debug("subscribe to event")

	secret := c.Param("secret")
	event, err := types.ConstructEvent(uri, secret, cronguru)
	if err != nil {
		log.WithError(err).Error("failed to construct event URI")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// add cron job
	err = runner.AddCronJob(*event)
	if err != nil {
		log.WithError(err).Error("failed to add cron job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

func unsubscribeFromEvent(c *gin.Context) {
	uri := getParam(c, "uri")
	log.WithField("uri (url-encoded)", uri).Debug("unsubscribe from event")
	// remove cron job
	err := runner.RemoveCronJob(uri)
	if err != nil {
		log.WithError(err).Error("failed to remove cron job")
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

// Ping return PONG with OK
func ping(c *gin.Context) {
	c.String(http.StatusOK, "PONG")
}

func backupDB(c *gin.Context) {
	size, err := store.BackupDB(c.Writer)
	if err != nil {
		log.WithError(err).Error("failed to backup BoltDB")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// set return header
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", `attachment; filename="my.db"`)
	c.Header("Content-Length", strconv.Itoa(size))
	c.Status(http.StatusOK)
}
