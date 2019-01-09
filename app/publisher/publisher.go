package publisher

import (
	"net/url"

	"github.com/ChimeraCoder/anaconda"
	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"

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
	v := url.Values{}
	v.Set("tweet_mode", "extended")
	msg := formatter(event)
	if _, err := api.PostTweet(msg, v); err != nil {
		return errors.Wrap(err, "can't send to twitter")
	}
	log.Printf("[DEBUG] published to twitter %s", msg)
	return nil
}
