package rss

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
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
	guid      string
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
		lastGUID := ""
		for {
			feedData, err := fp.ParseURL(n.Feed)
			if err != nil {
				log.Printf("[WARN] failed to fetch from %s, %s", n.Feed, err)
				if !waitOrCancel(n.ctx) {
					return
				}
				continue
			}
			event := n.feedEvent(feedData)
			if lastGUID != event.guid {
				if lastGUID != "" { // don't notify on initial change
					log.Printf("[DEBUG] new event %s", event.guid)
					ch <- event
				}
				lastGUID = event.guid
			}
			if !waitOrCancel(n.ctx) {
				return
			}
		}
	}()

	return ch
}

// Shutdown notifier
func (n *Notify) Shutdown() {
	n.cancel()
	<-n.ctx.Done()
}

func (n *Notify) feedEvent(feed *gofeed.Feed) (e Event) {
	e.ChanTitle = feed.Title
	if len(feed.Items) > 0 {
		e.Title = feed.Items[0].Title
		e.Link = feed.Items[0].Link
		e.guid = feed.Items[0].GUID
	}
	return e
}
