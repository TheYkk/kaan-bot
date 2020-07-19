package main

import (
	"context"
	"encoding/json"
	"fmt"
	ghub "kaan-bot/github"
	"kaan-bot/helper"
	"kaan-bot/plugins/label"
	"kaan-bot/plugins/title"
	"kaan-bot/types"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
)

var (
	Version   = "dev"
	GitCommit = "HEAD"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.JSONFormatter{})

	// Only log the warning severity or above.
	// log.SetLevel(log.WarnLevel)
}

func main() {
	log.Printf("Init kaan-bot %s %s ", Version, GitCommit)

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
	server.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	server.POST("/", func(c *gin.Context) {

		// * Get gihtub signature
		xHubSignature := c.GetHeader("X-Hub-Signature")


		// * Validate request

		rawData, _ := c.GetRawData()
		err := ghub.Validate(rawData, xHubSignature, secret)
		if err != nil {
			log.Fatal(err)
		}

		// ? Login with cred

		ctx := context.Background()
		client := ghub.Login(ctx, token)

		// ? Handle all events from github
		event := c.GetHeader("X-GitHub-Event")

		eventErr := handleEvent(client, event, rawData)
		if eventErr != nil {
			log.Error(eventErr)
		}

	})

	// ? listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	server.Run("0.0.0.0:8181")
}

//TTT
func handleEvent(gc *github.Client, eventType string, bytesIn []byte) error {

	switch eventType {
	case "release":

		break

	case "pull_request":

		break

	case "issue_comment", "issues":
		// * Parse event to req
		req := types.IssueCommentOuter{}
		if err := json.Unmarshal(bytesIn, &req); err != nil {
			return fmt.Errorf("Cannot parse input %s", err.Error())
		}
		var lines []string


		if req.Action == "opened" {
			lines = strings.Split(req.Issue.Body, "\n")
		} else {
			lines = strings.Split(req.Comment.Body, "\n")
		}

		// * Parse lines

		for _, line := range lines {

			log.Print(line)

			labelMatches := label.LabelRegex.FindAllStringSubmatch(line, -1)
			removeLabelMatches := label.RemoveLabelRegex.FindAllStringSubmatch(line, -1)
			customLabelMatches := label.CustomLabelRegex.FindAllStringSubmatch(line, -1)
			customRemoveLabelMatches := label.CustomRemoveLabelRegex.FindAllStringSubmatch(line, -1)

			// * If any match with regex sent to label handler
			if len(labelMatches) == 1 || len(removeLabelMatches) == 1 || len(customLabelMatches) == 1 || len(customRemoveLabelMatches) == 1 {
				err := label.Handle(gc, line, req)
				if err != nil {
					log.Error(err)
				}
			}

			retitleMatches := title.RetitleRegex.FindAllStringSubmatch(line, -1)
			if len(retitleMatches) == 1 {
				err := title.Handle(gc, line, req)
				if err != nil {
					log.Error(err)
				}
			}
		}

		break
	default:
		return fmt.Errorf("X_Github_Event: " + eventType)
	}

	return nil
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
