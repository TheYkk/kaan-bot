package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
	ghclient "kaan-bot/github"
	"kaan-bot/helper"
	"kaan-bot/plugins/label"
	"kaan-bot/plugins/lgtm"
	"kaan-bot/plugins/size"
	"kaan-bot/plugins/title"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)
// ? Version of build
var (
	Version = "dev"
)

var port = flag.String("port",helper.Getenv("PORT", strconv.Itoa(8181)),"Port to listen on for HTTP")
var printVersion = flag.Bool("v",false,"Print version")
var help = flag.Bool("help",false,"Get Help")
var listen = flag.String("listen",helper.Getenv("LISTEN", "0.0.0.0"), "IPv4 address to listen on")

func init() {

	flag.Usage = func() {
		flag.PrintDefaults()
		os.Exit(0)
	}
	flag.Parse()
	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *printVersion {
		fmt.Print(Version)
		os.Exit(0)
	}
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
	router := mux.NewRouter()

	router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"version": Version})
	}).Methods("GET")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "OK")
	}).Methods("GET")

	ctx := context.Background()
	// ? Login to github
	client := ghclient.Login(ctx, token)

	router.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		hook, _ := webhook.New(webhook.Options.Secret(secret))
		payload, err := hook.Parse(r, webhook.ReleaseEvent, webhook.PullRequestEvent, webhook.IssueCommentEvent)
		if err != nil {
			if err == webhook.ErrEventNotFound {
				log.Error(err)
			}
		}

		switch payload.(type) {

		case webhook.PullRequestPayload:
			log.Info("Is pull request")
			pullRequest := payload.(webhook.PullRequestPayload)

			err := size.Handle(client, pullRequest)
			if err != nil {
				log.Error(err)
			}

			// ? Remove labels
			errLabel := lgtm.RemoveLabel(client, pullRequest)
			if err != nil {
				log.Error(errLabel)
			}
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
				lgtmMatches := lgtm.LGTMRe.FindAllStringSubmatch(line, -1)
				lgtmcancelMatches := lgtm.LGTMCancelRe.FindAllStringSubmatch(line, -1)

				if len(lgtmMatches) == 1 || len(lgtmcancelMatches) == 1 {
					err := lgtm.Handle(client, line, comment)
					if err != nil {
						log.Error(err)
					}
				}
				// TODO: assign
			}
		}

		_, _ = fmt.Fprint(w, "Event received. Have a nice day")
	}).Methods("POST")

	// ? listen and serve on default 0.0.0.0:8181
	srv := &http.Server{
		Handler: tracing()(logging()(router)),
		Addr:    *listen + ":" + *port,
		// ! Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Info("Serve at: ",*listen + ":" + *port)
	//log.Fatal(srv.ListenAndServe())
	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Server err %s", err)
	}


}
type key int

const (
	requestIDKey key = 0
)

var timeFormat = "2006-01-02T15:04:05-07:00"

func logging() func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			requestID, ok := r.Context().Value(requestIDKey).(string)
			if !ok {
				requestID = "unknown"
			}

			hostname, err := os.Hostname()
			if err != nil {
				hostname = "unknow"
			}

			start := time.Now()

			next.ServeHTTP(w, r)

			stop := time.Since(start)
			latency := int(math.Ceil(float64(stop.Nanoseconds()) / 1000000.0))

			IPAddress := r.Header.Get("X-Real-Ip")
			if IPAddress == "" {
				IPAddress = r.Header.Get("X-Forwarded-For")
			}
			if IPAddress == "" {
				IPAddress = r.RemoteAddr
			}

			log.WithFields(log.Fields{
				"hostname":   hostname,
				"requestID":   requestID,
				"latency":    latency, // time to process
				"clientIP":   IPAddress,
				"method":     r.Method,
				"path":       r.URL.Path,
				"referer":    r.Referer(),
				"userAgent":  r.UserAgent(),
			}).Info("Request")
		})
	}
}
func tracing() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = fmt.Sprintf("%d", time.Now().UnixNano())
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
