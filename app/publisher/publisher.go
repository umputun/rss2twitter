package publisher

import (
	"net/url"

	"github.com/ChimeraCoder/anaconda"
	log "github.com/go-pkgz/lgr"

	"github.com/umputun/rss2twitter/app/rss"
)

// Interface for publishers
type Interface interface {
	Publish(event rss.Event, formatter func(rss.Event) string) error
}

// Stdout implements publisher.Interface and sends to stdout
type Stdout struct{}

// Publish to logger
func (s Stdout) Publish(event rss.Event, formatter func(rss.Event) string) error {
	log.Printf("[INFO] event - %s", formatter(event))
	return nil
}

// Twitter implements publisher.Interface and sends to twitter
type Twitter struct {
	ConsumerKey, ConsumerSecret string
	AccessToken, AccessSecret   string
}

// Publish to twitter
func (t Twitter) Publish(event rss.Event, formatter func(rss.Event) string) error {
	log.Printf("[INFO] publish to twitter %+v", event)
	api := anaconda.NewTwitterApiWithCredentials(t.AccessToken, t.AccessSecret, t.ConsumerKey, t.ConsumerSecret)
	_, err := api.PostTweet(formatter(event), url.Values{})
	return err
}
