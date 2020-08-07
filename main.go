package main

import (
	"context"
	"fmt"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
	ghclient "kaan-bot/github"
	"kaan-bot/helper"
	"kaan-bot/plugins/label"
	"kaan-bot/plugins/title"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var (
	Version = "dev"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetReportCaller(true)
}

func main() {
	log.Printf("Init kaan-bot %s", Version)

	secret := helper.Getenv("GITHUB_SECRET", "")
	if secret == "" {
		log.Fatal("Github webhook secret not set")
		return
	}

	token := helper.Getenv("GITHUB_TOKEN", "")
	if token == "" {
		log.Error("Github token not set")

	}

	// ? Create http server
	server := gin.Default()

	// ? Set logger to logrus
	server.Use(Logger(log.New()), gin.Recovery())

	server.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version": Version,
		})
	})

	server.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	server.POST("/", func(c *gin.Context) {

		hook, _ := webhook.New(webhook.Options.Secret(secret))
		payload, err := hook.Parse(c.Request, webhook.ReleaseEvent, webhook.PullRequestEvent, webhook.IssueCommentEvent)
		if err != nil {
			if err == webhook.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}

		ctx := context.Background()
		// ? Login to github
		client := ghclient.Login(ctx, token)

		switch payload.(type) {

		case webhook.PullRequestPayload:
			pullRequest := payload.(webhook.PullRequestPayload)
			// Do whatever you want from here...
			fmt.Printf("%+v", pullRequest.Repository.FullName)

			// TODO: Size plugin
			// TODO: DCO

		case webhook.IssueCommentPayload:
			comment := payload.(webhook.IssueCommentPayload)
			lines := strings.Split(comment.Comment.Body, "\n")

			// * Parse lines
			for _, line := range lines {

				log.Print(line)

				labelMatches := label.LabelRegex.FindAllStringSubmatch(line, -1)
				removeLabelMatches := label.RemoveLabelRegex.FindAllStringSubmatch(line, -1)
				customLabelMatches := label.CustomLabelRegex.FindAllStringSubmatch(line, -1)
				customRemoveLabelMatches := label.CustomRemoveLabelRegex.FindAllStringSubmatch(line, -1)

				// * If any match with regex sent to label handler
				if len(labelMatches) == 1 || len(removeLabelMatches) == 1 || len(customLabelMatches) == 1 || len(customRemoveLabelMatches) == 1 {
					err := label.Handle(client, line, comment)
					if err != nil {
						log.Error(err)
					}
				}
				// * If any match with regex sent to title handler
				retitleMatches := title.RetitleRegex.FindAllStringSubmatch(line, -1)
				if len(retitleMatches) == 1 {
					err := title.Handle(client, line, comment)
					if err != nil {
						log.Error(err)
					}
				}

				// TODO: lgtm

				// TODO: assign
			}
		}

		// ? Login with cred

	})

	// ? listen and serve on 0.0.0.0:8181
	err := server.Run("0.0.0.0:8181")
	if err != nil {
		log.Fatalf("Server err %s", err)
	}
}

var timeFormat = "02/Jan/2006:15:04:05 -0700"

func Logger(logger log.FieldLogger) gin.HandlerFunc {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknow"
	}
	return func(c *gin.Context) {
		// other handler can change c.Path so:
		path := c.Request.URL.Path
		start := time.Now()
		c.Next()
		stop := time.Since(start)
		latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		clientUserAgent := c.Request.UserAgent()
		referer := c.Request.Referer()
		dataLength := c.Writer.Size()
		if dataLength < 0 {
			dataLength = 0
		}

		entry := log.WithFields(log.Fields{
			"hostname":   hostname,
			"statusCode": statusCode,
			"latency":    latency, // time to process
			"clientIP":   clientIP,
			"method":     c.Request.Method,
			"path":       path,
			"referer":    referer,
			"dataLength": dataLength,
			"userAgent":  clientUserAgent,
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.ByType(gin.ErrorTypePrivate).String())
		} else {
			msg := fmt.Sprintf("%s - %s [%s] \"%s %s\" %d %d \"%s\" \"%s\" (%dms)", clientIP, hostname, time.Now().Format(timeFormat), c.Request.Method, path, statusCode, dataLength, referer, clientUserAgent, latency)
			if statusCode > 499 {
				entry.Error(msg)
			} else if statusCode > 399 {
				entry.Warn(msg)
			} else {
				entry.Info(msg)
			}
		}
	}
}
