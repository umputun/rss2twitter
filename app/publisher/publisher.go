package publisher

import (
	"log"
	"net/url"
	"time"

	"github.com/ChimeraCoder/anaconda"
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
	log.Printf("[EVENT] %s", formatter(event))
	return nil
}

// Twitter implements publisher.Interface and sends to twitter
type Twitter struct {
	ConsumerKey, ConsumerSecret string
	AccessToken, AccessSecret   string
}

// Publish to twitter
func (t Twitter) Publish(event rss.Event, formatter func(rss.Event) string) error {
	api := anaconda.NewTwitterApiWithCredentials(t.AccessToken, t.AccessSecret, t.ConsumerKey, t.ConsumerSecret)
	api.SetDelay(5 * time.Second)
	_, err := api.PostTweet(formatter(event), url.Values{})
	return err
}
