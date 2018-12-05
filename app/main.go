package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/umputun/rss2twitter/app/publisher"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
	"github.com/umputun/rss2twitter/app/rss"
)

var opts struct {
	Refresh time.Duration `short:"r" long:"refresh" env:"REFRESH" default:"30s" description:"refresh interval"`
	TimeOut time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"5s" description:"twitter timeout"`
	Feed    string        `short:"f" long:"feed" env:"FEED" required:"true" description:"rss feed url"`

	ConsumerKey    string `long:"consumer-key" env:"CONSUMER_KEY" required:"true" description:"twitter consumer key"`
	ConsumerSecret string `long:"consumer-secret" env:"CONSUMER_SECRET" required:"true" description:"twitter consumer secret"`
	AccessToken    string `long:"access-token" env:"ACCESS_TOKEN" required:"true" description:"twitter access token"`
	AccessSecret   string `long:"access-secret" env:"ACCESS_SECRET" required:"true" description:"twitter access secret"`

	Template string `long:"template" env:"TEMPLATE" default:"{{.Title}} - {{.Link}}" description:"twitter message template"`
	Dbg      bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("RSS2TWITTER - %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	setupLog(opts.Dbg)

	notifier := rss.New(context.Background(), opts.Feed, opts.Refresh)
	pub := publisher.Twitter{
		ConsumerKey:    opts.ConsumerKey,
		ConsumerSecret: opts.ConsumerSecret,
		AccessToken:    opts.AccessToken,
		AccessSecret:   opts.AccessSecret,
	}

	for event := range notifier.Go() {
		err := pub.Publish(event, func(r rss.Event) string {
			b1 := bytes.Buffer{}
			if err := template.Must(template.New("twi").Parse(opts.Template)).Execute(&b1, event); err != nil {
				return fmt.Sprintf("%s - %s", r.Title, r.Link)
			}
			return b1.String()
		})
		if err != nil {
			log.Printf("[WARN] failed to publish, %s", err)
		}
	}
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
