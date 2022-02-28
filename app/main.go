package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/denisbrodbeck/striphtmltags"
	log "github.com/go-pkgz/lgr"
	"github.com/umputun/go-flags"

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

	catchSignals()

	notif, pub, err := setup(o)
	if err != nil {
		log.Printf("[PANIC] failed to setup, %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch SIGTERM signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Printf("[WARN] interrupt signal")
		cancel()
	}()

	do(ctx, notif, pub, o.Template)
	log.Print("[INFO] terminated")
}

func setup(o opts) (n notifier, p publisher.Interface, err error) {
	content, err := os.ReadFile("exclusion-patterns.txt")
	if err != nil {
		log.Printf("[WARN] could not read 'exclusion-patterns.txt' file: %v", err)
		content = []byte{}
	}
	lines := strings.Split(string(content), "\n")
	n = &rss.Notify{Feed: o.Feed, Duration: o.Refresh, Timeout: o.TimeOut}
	p = publisher.Twitter{
		ConsumerKey:    o.ConsumerKey,
		ConsumerSecret: o.ConsumerSecret,
		AccessToken:    o.AccessToken,
		AccessSecret:   o.AccessSecret,
		ExcludeList:    lines,
	}

	if o.Dry { // override publisher to stdout only, no actual twitter publishing
		p = publisher.Stdout{
			ExcludeList: lines,
		}
		log.Print("[INFO] dry mode")
	}

	if !o.Dry && (o.ConsumerKey == "" || o.ConsumerSecret == "" || o.AccessToken == "" || o.AccessSecret == "") {
		return n, p, errors.New("token credentials missing")
	}
	return n, p, nil
}

// do runs event loop getting rss events, formatting and publishing them
func do(ctx context.Context, notif notifier, pub publisher.Interface, tmpl string) {
	log.Printf("[INFO] message template - %q", tmpl)
	ch := notif.Go(ctx)
	for event := range ch {
		err := pub.Publish(event, func(r rss.Event) string { return formatMsg(event, tmpl, 279) })
		if err != nil {
			log.Printf("[WARN] failed to publish, %s", err)
		}
	}
}

// formatMsg makes a tweet message from rss event, strip html tags and shorten text if necessary
func formatMsg(ev rss.Event, tmpl string, max int) string {

	shortLinkLen := 23 // url of any length altered to 23 characters by twitter, even if the link itself is less than 23

	// strip html tags from title and text
	ev.Title = striphtmltags.StripTags(ev.Title)
	ev.Text = striphtmltags.StripTags(ev.Text)

	trimWithDots := func(s string, max int) string {
		if len([]rune(s)) <= max || max < 4 {
			return s
		}
		snippet := []rune(s)[:max-4] // extra 4 for dots
		// go back in snippet and found the first space to trim nicely, on the word boundary
		for i := len(snippet) - 1; i >= 0; i-- {
			if snippet[i] == ' ' {
				snippet = snippet[:i]
				break
			}
		}
		return string(snippet) + "... " // extra space at the end to make it look better if it has something after
	}

	applyTempl := func(ev rss.Event, tmpl string) string {
		var res string
		b1 := bytes.Buffer{}
		if err := template.Must(template.New("twi").Parse(tmpl)).Execute(&b1, ev); err != nil { // nolint
			// template failed to parse record, backup with predefined format
			res = trimWithDots(fmt.Sprintf("%s - %s", ev.Title, ev.Link), max)
		} else {
			res = b1.String()
		}
		return strings.Replace(res, `\n`, "\n", -1) // handle \n we may have in the template
	}

	// if no Link in rss.Event, just apply template and trim resulted message directly
	if !strings.Contains(tmpl, "{{.Link}}") {
		return trimWithDots(applyTempl(ev, tmpl), max)
	}

	// remove all template elements to get the len of the message without it
	// this is needed to calculate the length of the constants parts of the message,
	// i.e. for "{{.Title}} blah {{.Link}}" it is 6, len(" blah ")
	noTmpl := tmpl
	for _, t := range []string{"{{.Link}}", "{{.Title}}", "{{.Text}}"} {
		noTmpl = strings.Replace(noTmpl, t, "", -1)
	}
	noTmplLen := len([]rune(noTmpl))

	textOrTitleMax := max - shortLinkLen - noTmplLen
	switch {
	case strings.Contains(tmpl, "{{.Text}}"): // first trim text, if in template
		ev.Text = trimWithDots(ev.Text, textOrTitleMax)
	case strings.Contains(tmpl, "{{.Title}}"): // if not, trim title if in template
		ev.Title = trimWithDots(ev.Title, textOrTitleMax)
	}

	// apply template with altered event values.
	return applyTempl(ev, tmpl)
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

func catchSignals() {
	// catch SIGQUIT and print stack traces
	sigChan := make(chan os.Signal, 1)
	go func() {
		for range sigChan {
			log.Printf("[INFO] SIGQUIT detected, dump:\n%s", getDump())
		}
	}()
	signal.Notify(sigChan, syscall.SIGQUIT)
}
