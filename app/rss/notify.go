package rss

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/mmcdole/gofeed"
)

// Notify on RSS change
type Notify struct {
	feed     string
	duration time.Duration

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

// New makes notifier for given rss feed. Checks for new items every duration
func New(ctx context.Context, feed string, duration time.Duration) *Notify {
	res := Notify{feed: feed, duration: duration}
	res.ctx, res.cancel = context.WithCancel(ctx)
	return &res
}

// Go starts notifier and returns events channel
func (n *Notify) Go() <-chan Event {
	ch := make(chan Event)
	go func() {
		defer func() {
			close(ch)
			n.cancel()
		}()
		fp := gofeed.NewParser()
		fp.Client = &http.Client{Timeout: time.Second * 5}
		lastGUID := ""
		for {
			feedData, err := fp.ParseURL(n.feed)
			if err != nil {
				log.Printf("[WARN] failed to fetch from %s, %s", n.feed, err)
				time.Sleep(n.duration)
				continue
			}
			event := n.feedEvent(feedData)
			if lastGUID != event.guid {
				if lastGUID != "" {
					ch <- event
				}
				lastGUID = event.guid
			}
			select {
			case <-n.ctx.Done():
				return
			case <-time.After(n.duration):
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
