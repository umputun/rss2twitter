// Package publisher sends forward rss events to publisher interface (twitter)
package publisher

import (
	"net/url"
	"regexp"
	"strings"

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
type Stdout struct{
	ExcludeList []string
}

// CheckExclusionList checks the exclusion list for matches
func CheckExclusionList(excludes []string, msg string) bool {
	for _, value := range excludes {
		if len(value) > 0 && !strings.HasPrefix(value, "#") {
			if match, _ := regexp.MatchString(strings.ToLower(value), strings.ToLower(msg)); match {
				log.Printf("[EXCLUDED] matched: %s - %s", value, msg)
				return true
			}
		}
	}

	return false
}

// Publish to logger
func (s Stdout) Publish(event rss.Event, formatter func(rss.Event) string) error {
	msg := formatter(event)
	if CheckExclusionList(s.ExcludeList, msg) {
		return nil
	}
	log.Printf("[INFO] event - %s", msg)
	return nil
}

// Twitter implements publisher.Interface and sends to twitter
type Twitter struct {
	ConsumerKey, ConsumerSecret string
	AccessToken, AccessSecret   string
	ExcludeList []string
}

// Publish to twitter
func (t Twitter) Publish(event rss.Event, formatter func(rss.Event) string) error {
	log.Printf("[INFO] publish to twitter %+v", event.Title)
	api := anaconda.NewTwitterApiWithCredentials(t.AccessToken, t.AccessSecret, t.ConsumerKey, t.ConsumerSecret)
	v := url.Values{}
	v.Set("tweet_mode", "extended")
	msg := formatter(event)
	// See if it's been excluded
	if CheckExclusionList(t.ExcludeList, msg) {
		return nil
	}
	// Post the message
	if _, err := api.PostTweet(msg, v); err != nil {
		return errors.Wrap(err, "can't send to twitter")
	}
	log.Printf("[DEBUG] published to twitter %s", strings.Replace(msg, "\n", " ", -1))
	return nil
}
