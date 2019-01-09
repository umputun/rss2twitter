package main

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/umputun/rss2twitter/app/rss"
)

func TestSetupDry(t *testing.T) {
	o := opts{Feed: "http://example.com", Dry: true}
	n, p, err := setup(o)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, "publisher.Stdout", fmt.Sprintf("%T", p))
}

func TestSetupFull(t *testing.T) {
	o := opts{Feed: "http://example.com", Dry: false,
		ConsumerKey: "1", ConsumerSecret: "1", AccessToken: "1", AccessSecret: "1"}
	n, p, err := setup(o)
	require.NoError(t, err)
	assert.NotNil(t, n)
	assert.Equal(t, "publisher.Twitter", fmt.Sprintf("%T", p))
}

func TestSetupFailed(t *testing.T) {
	o := opts{Feed: "http://example.com", Dry: false,
		ConsumerKey: "1", ConsumerSecret: "1"}
	_, _, err := setup(o)
	assert.NotNil(t, err)
}

func TestDo(t *testing.T) {
	pub := pubMock{buf: bytes.Buffer{}}
	notif := notifierMock{delay: 100 * time.Millisecond, events: []rss.Event{
		{GUID: "1", Title: "t1", Link: "l1", Text: "ttt2"},
		{GUID: "2", Title: "t2", Link: "l2", Text: "ttt2"},
		{GUID: "3", Title: "t4", Link: "l3", Text: "ttt3"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	do(ctx, &notif, &pub, "{{.Title}} - {{.Link}}")
	cancel()
	assert.Equal(t, "t1 - l1\nt2 - l2\nt4 - l3\n", pub.buf.String())
}

func TestDoCanceled(t *testing.T) {
	pub := pubMock{buf: bytes.Buffer{}}
	notif := notifierMock{delay: 100 * time.Millisecond, events: []rss.Event{
		{GUID: "1", Title: "t1", Link: "l1", Text: "ttt2"},
		{GUID: "2", Title: "t2", Link: "l2", Text: "ttt2"},
		{GUID: "3", Title: "t4", Link: "l3", Text: "ttt3"},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(time.Millisecond*150, func() { cancel() })
	do(ctx, &notif, &pub, "{{.Title}} - {{.Link}} {{.Text}}")
	assert.Equal(t, "t1 - l1 ttt2\n", pub.buf.String())
}

func TestFormat(t *testing.T) {
	tbl := []struct {
		inp  string
		out  string
		size int
	}{
		{"blah", "blah", 100},
		{"blah <p>xyx</p>", "blah xyx", 100},
		{"blah <p>xyx</p> something 122 abcdefg 12345 qwer", "blah xyx ...", 15},
		{"blah <p>xyx</p> something 122 abcdefg 12345 qwerty", "blah xyx something ...", 20},
		{"<p>xyx</p><title>", "xyx", 20},
	}

	for i, tt := range tbl {
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			out := format(tt.inp, tt.size)
			assert.Equal(t, tt.out, out)
		})
	}
}

type pubMock struct {
	buf bytes.Buffer
}

func (m *pubMock) Publish(event rss.Event, formatter func(rss.Event) string) error {
	_, err := m.buf.WriteString(formatter(event) + "\n")
	return err
}

type notifierMock struct {
	events []rss.Event
	delay  time.Duration
}

func (m *notifierMock) Go(ctx context.Context) <-chan rss.Event {
	ch := make(chan rss.Event)
	go func() {
		for _, e := range m.events {
			select {
			case <-ctx.Done():
				break
			case <-time.After(m.delay):
				ch <- e
			}
		}
		close(ch)
	}()
	return ch
}
