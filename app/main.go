package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"text/template"
	"time"

	log "github.com/go-pkgz/lgr"
	flags "github.com/jessevdk/go-flags"

	"github.com/umputun/rss2twitter/app/publisher"
	"github.com/umputun/rss2twitter/app/rss"
)

var opts struct {
	Refresh time.Duration `short:"r" long:"refresh" env:"REFRESH" default:"30s" description:"refresh interval"`
	TimeOut time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"5s" description:"rss feed timeout"`
	Feed    string        `short:"f" long:"feed" env:"FEED" required:"true" description:"rss feed url"`

	ConsumerKey    string `long:"consumer-key" env:"TWI_CONSUMER_KEY" required:"true" description:"twitter consumer key"`
	ConsumerSecret string `long:"consumer-secret" env:"TWI_CONSUMER_SECRET" required:"true" description:"twitter consumer secret"`
	AccessToken    string `long:"access-token" env:"TWI_ACCESS_TOKEN" required:"true" description:"twitter access token"`
	AccessSecret   string `long:"access-secret" env:"TWI_ACCESS_SECRET" required:"true" description:"twitter access secret"`

	Template string `long:"template" env:"TEMPLATE" default:"{{.Title}} - {{.Link}}" description:"twitter message template"`
	Dry      bool   `long:"dry" env:"DRY" description:"dry mode"`
	Dbg      bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("rss2twitter - %s\n", revision)
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	if opts.Dbg {
		log.Setup(log.Debug)
	}

	notifier := rss.Notify{Feed: opts.Feed, Duration: opts.Refresh, Timeout: opts.TimeOut}
	var pub publisher.Interface = publisher.Twitter{
		ConsumerKey:    opts.ConsumerKey,
		ConsumerSecret: opts.ConsumerSecret,
		AccessToken:    opts.AccessToken,
		AccessSecret:   opts.AccessSecret,
	}

	if opts.Dry { // override publisher to stdout only, no actual twitter publishing
		pub = publisher.Stdout{}
		log.Print("[INFO] dry mode")
	}

	log.Printf("[INFO] message template - %q", opts.Template)

	ch := notifier.Go(context.Background())
	for event := range ch {
		err := pub.Publish(event, func(r rss.Event) string {
			b1 := bytes.Buffer{}
			if err := template.Must(template.New("twi").Parse(opts.Template)).Execute(&b1, event); err != nil {
				// template failed to parse record, backup predefined format
				return fmt.Sprintf("%s - %s", r.Title, r.Link)
			}
			return b1.String()
		})
		if err != nil {
			log.Printf("[WARN] failed to publish, %s", err)
		}
	}
	log.Print("[INFO] terminated")
}

// getDump reads runtime stack and returns as a string
func getDump() string {
	maxSize := 5 * 1024 * 1024
	stacktrace := make([]byte, maxSize)
	length := runtime.Stack(stacktrace, true)
	if length > maxSize {
		length = maxSize
	}
	return string(stacktrace[:length])
}

func init() {
	// catch SIGQUIT and print stack traces
	sigChan := make(chan os.Signal)
	go func() {
		for range sigChan {
			log.Printf("[INFO] SIGQUIT detected, dump:\n%s", getDump())
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT)
}
