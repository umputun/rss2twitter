// Package rss implements rss event publisher. Posts new events to returned channel.
// Each event represents a new rss item
package rss

import (
	"context"
	"net/http"
	"sync"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/mmcdole/gofeed"
	"github.com/pkg/errors"
)

// Notify on RSS change
type Notify struct {
	Feed     string
	Duration time.Duration
	Timeout  time.Duration

	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
}

// Event from RSS
type Event struct {
	ChanTitle string
	Title     string
	Link      string
	Text      string
	GUID      string
}

// Go starts notifier and returns events channel
func (n *Notify) Go(ctx context.Context) <-chan Event {
	log.Printf("[INFO] start notifier for %s, every %s", n.Feed, n.Duration)
	n.once.Do(func() { n.ctx, n.cancel = context.WithCancel(ctx) })

	ch := make(chan Event)

	// wait for duration, can be terminated by ctx
	waitOrCancel := func(ctx context.Context) bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(n.Duration):
			return true
		}
	}

	go func() {

		defer func() {
			close(ch)
			n.cancel()
		}()

		fp := gofeed.NewParser()
		fp.Client = &http.Client{Timeout: n.Timeout}
		log.Printf("[DEBUG] notifier uses http timeout %v", n.Timeout)
		lastGUID := ""
		for {
			feedData, err := fp.ParseURL(n.Feed)
			if err != nil {
				log.Printf("[WARN] failed to fetch/parse url from %s, %v", n.Feed, err)
				if !waitOrCancel(n.ctx) {
					return
				}
				continue
			}
			event, err := n.feedEvent(feedData)
			if lastGUID != event.GUID && err == nil {
				if lastGUID != "" { // don't notify on initial change
					log.Printf("[INFO] new event %s - %s", event.GUID, event.Title)
					ch <- event
				} else {
					log.Printf("[INFO] ignore first event %s - %s", event.GUID, event.Title)
				}
				lastGUID = event.GUID
			}
			if !waitOrCancel(n.ctx) {
				log.Print("[WARN] notifier canceled")
				return
			}
		}
	}()

	return ch
}

// Shutdown notifier
func (n *Notify) Shutdown() {
	log.Print("[DEBUG] shutdown initiated")
	n.cancel()
	<-n.ctx.Done()
}

// feedEvent gets latest item from rss feed
func (n *Notify) feedEvent(feed *gofeed.Feed) (e Event, err error) {
	if len(feed.Items) == 0 {
		return e, errors.New("no items in rss feed")
	}
	if feed.Items[0].GUID == "" {
		return e, errors.Errorf("no guid for rss entry %+v", feed.Items[0])
	}

	e.ChanTitle = feed.Title
	e.Title = feed.Items[0].Title
	e.Link = feed.Items[0].Link
	e.Text = feed.Items[0].Description
	e.GUID = feed.Items[0].GUID

	return e, nil
}
