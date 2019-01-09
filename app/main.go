package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"text/template"
	"time"

	"github.com/denisbrodbeck/striphtmltags"
	log "github.com/go-pkgz/lgr"
	flags "github.com/jessevdk/go-flags"

	"github.com/umputun/rss2twitter/app/publisher"
	"github.com/umputun/rss2twitter/app/rss"
)

type opts struct {
	Refresh time.Duration `short:"r" long:"refresh" env:"REFRESH" default:"30s" description:"refresh interval"`
	TimeOut time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"5s" description:"rss feed timeout"`
	Feed    string        `short:"f" long:"feed" env:"FEED" required:"true" description:"rss feed url"`

	ConsumerKey    string `long:"consumer-key" env:"TWI_CONSUMER_KEY" description:"twitter consumer key"`
	ConsumerSecret string `long:"consumer-secret" env:"TWI_CONSUMER_SECRET" description:"twitter consumer secret"`
	AccessToken    string `long:"access-token" env:"TWI_ACCESS_TOKEN" description:"twitter access token"`
	AccessSecret   string `long:"access-secret" env:"TWI_ACCESS_SECRET" description:"twitter access secret"`

	Template string `long:"template" env:"TEMPLATE" default:"{{.Title}} - {{.Link}}" description:"twitter message template"`
	Dry      bool   `long:"dry" env:"DRY" description:"dry mode"`
	Dbg      bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

type notifier interface {
	Go(ctx context.Context) <-chan rss.Event
}

func main() {
	fmt.Printf("rss2twitter - %s\n", revision)
	o := opts{}
	if _, err := flags.Parse(&o); err != nil {
		os.Exit(1)
	}

	if o.Dbg {
		log.Setup(log.Debug)
	}

	notif, pub, err := setup(o)
	if err != nil {
		log.Printf("[PANIC] failed to setup, %v", err)
	}

	do(context.Background(), notif, pub, o.Template)

	log.Print("[INFO] terminated")
}

func setup(o opts) (n notifier, p publisher.Interface, err error) {
	n = &rss.Notify{Feed: o.Feed, Duration: o.Refresh, Timeout: o.TimeOut}
	p = publisher.Twitter{
		ConsumerKey:    o.ConsumerKey,
		ConsumerSecret: o.ConsumerSecret,
		AccessToken:    o.AccessToken,
		AccessSecret:   o.AccessSecret,
	}

	if o.Dry { // override publisher to stdout only, no actual twitter publishing
		p = publisher.Stdout{}
		log.Print("[INFO] dry mode")
	} else {
		if o.ConsumerKey == "" || o.ConsumerSecret == "" || o.AccessToken == "" || o.AccessSecret == "" {
			return n, p, errors.New("token credentials missing")
		}
	}
	return n, p, nil
}

// do runs event loop getting rss events and publishing them
func do(ctx context.Context, notif notifier, pub publisher.Interface, tmpl string) {
	log.Printf("[INFO] message template - %q", tmpl)
	ch := notif.Go(ctx)
	for event := range ch {
		err := pub.Publish(event, func(r rss.Event) string {
			b1 := bytes.Buffer{}
			if err := template.Must(template.New("twi").Parse(tmpl)).Execute(&b1, event); err != nil {
				// template failed to parse record, backup predefined format
				return fmt.Sprintf("%s - %s", r.Title, r.Link)
			}
			return format(b1.String(), 275)
		})
		if err != nil {
			log.Printf("[WARN] failed to publish, %s", err)
		}
	}
}

// format cleans text (removes html tags) and shrinks result
func format(inp string, max int) string {
	res := striphtmltags.StripTags(inp)
	if len([]rune(res)) > max {
		snippet := []rune(res)[:max]
		//go back in snippet and found first space
		for i := len(snippet) - 1; i >= 0; i-- {
			if snippet[i] == ' ' {
				snippet = snippet[:i]
				break
			}
		}
		res = string(snippet) + " ..."
	}
	return res
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
