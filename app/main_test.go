package main

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/rss2twitter/app/publisher"
	"github.com/umputun/rss2twitter/app/rss"
)

func TestMainApp(t *testing.T) {

	var n int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("req %+v", r)
		fnum := atomic.AddInt32(&n, int32(1))
		if fnum > 2 {
			fnum = 2
		}
		data, err := os.ReadFile(fmt.Sprintf("rss/testdata/f%d.xml", fnum))
		require.NoError(t, err)
		w.WriteHeader(200)
		_, _ = w.Write(data)
	}))
	defer ts.Close()

	os.Args = []string{"app", "--feed=" + ts.URL + "/rss", "--dry", "--dbg", "--refresh=100ms"}

	go func() {
		time.Sleep(500 * time.Millisecond)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.Nil(t, err, "kill")
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		st := time.Now()
		main()
		require.True(t, time.Since(st).Seconds() < 1, "should take about 500msec")
		wg.Done()
	}()
	wg.Wait()
}
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
		{GUID: "4", Title: "t5", Link: "http://example.com", Text: "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores "},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	do(ctx, &notif, &pub, "{{.Title}} - {{.Link}}")
	cancel()
	assert.Equal(t, "t1 - l1\nt2 - l2\nt4 - l3\nt5 - http://example.com\n", pub.buf.String())
}

func TestDoWithText(t *testing.T) {
	pub := pubMock{buf: bytes.Buffer{}}
	notif := notifierMock{delay: 100 * time.Millisecond, events: []rss.Event{
		{GUID: "1", Title: "t1", Link: "l1", Text: "ttt2"},
		{GUID: "2", Title: "t2", Link: "l2", Text: "ttt2"},
		{GUID: "3", Title: "t4", Link: "l3", Text: "ttt3"},
		{GUID: "4", Title: "t5", Link: "http://example.com", Text: "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores "},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	do(ctx, &notif, &pub, "{{.Text}} - {{.Link}}")
	cancel()
	assert.Equal(t, "ttt2 - l1\nttt2 - l2\nttt3 - l3\nLorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores  - http://example.com\n", pub.buf.String())
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

func Test_formatMsg(t *testing.T) {
	tbl := []struct {
		inp  rss.Event
		tmpl string
		max  int
		res  string
	}{
		{rss.Event{Text: "test", Link: "link"}, "{{.Text}} :: {{.Link}} 12345", 100, "test :: link 12345"},
		{rss.Event{Text: "test too long to fit to a <a href=blah>tweet</a>", Link: "link5678901234567890123"},
			"{{.Text}} :: {{.Link}}",
			50,
			"test too long to...  :: link5678901234567890123",
		},
		{
			rss.Event{Text: "test too long to fit to a tweet", Link: "link5678901234567890123"},
			"12345 {{.Link}} xxx {{.Text}} yes",
			50,
			"12345 link5678901234567890123 xxx test...  yes",
		},
		{
			rss.Event{Text: "test too long to fit to a <a href=example.com>tweet</a>"},
			"12345 xxx blah blah {{.Text}} \n hmm 123456",
			50,
			"12345 xxx blah blah test too long to fit to a... ",
		},
		{
			rss.Event{Text: `test <a href="https://example.com">ok\n yes?`, Link: "link5678901234567890123"},
			"{{.Text}} :: {{.Link}}",
			50,
			"test ok\n yes? :: link5678901234567890123",
		},
		{
			rss.Event{Title: "test too long to fit to a tweet", Link: "link5678901234567890123"},
			"12345 {{.Link}} xxx {{.Title}} \n yes",
			50,
			"12345 link5678901234567890123 xxx test...  \n yes",
		},
		{
			rss.Event{Title: "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores       Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126 sed diam voluptua. At vero eos et accusam et justo duo dolores", Link: "https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126"},
			"{{.Title}} - {{.Link}}",
			279,
			"Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores       Lorem ipsum dolor sit amet, consetetur...  - https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126",
		},
		{
			rss.Event{Title: "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores       Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores", Link: "https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126"},
			"{{.Title}} - {{.Link}}",
			279,
			"Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores       Lorem ipsum dolor sit amet, consetetur...  - https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126",
		},
		{
			rss.Event{Title: "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores       Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores", Link: "https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126"},
			"{{.Link}} - {{.Title}}",
			279,
			"https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126 - Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores       Lorem ipsum dolor sit amet, consetetur... ",
		},
		{
			rss.Event{Title: "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores", Link: "https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126"},
			"{{.Title}} - {{.Link}}",
			279,
			"Lorem ipsum dolor sit amet, consetetur sadipscing elitr, sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat, sed diam voluptua. At vero eos et accusam et justo duo dolores - https://github.com/umputun/rss2twitter/blob/d5c89112e4eb8ed8d1d0717526804bc145202fe5/app/main.go#L126",
		},
	}

	for i, tt := range tbl {
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			res := formatMsg(tt.inp, tt.tmpl, tt.max)
			assert.Equal(t, tt.res, res)
			t.Logf("res len: %d", len(res))
		})
	}
}

func TestExclusionPatterns(t *testing.T) {
	excludes := []string{
		"^The",
		"end$",
		"^The end$",
		"roar",
	}
	tbl := []struct {
		msg    string
		result bool
	}{
		{"The end of the world", true},
		{"This is the end", true},
		{"The end", true},
		{"Hear the mighty roar of the lion", true},
		{"You shall pass!", false},
	}

	for i, tt := range tbl {
		t.Run(fmt.Sprintf("check-%d", i), func(t *testing.T) {
			result := publisher.CheckExclusionList(excludes, tt.msg)
			assert.Equal(t, tt.result, result)
		})
	}
}

func TestGetDump(t *testing.T) {
	dump := getDump()
	assert.True(t, strings.Contains(dump, "goroutine"))
	assert.True(t, strings.Contains(dump, "[running]"))
	assert.True(t, strings.Contains(dump, "app/main.go"))
	log.Printf("\n dump: %s", dump)
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
		defer close(ch)
		for _, e := range m.events {
			select {
			case <-ctx.Done():
				return
			case <-time.After(m.delay):
				ch <- e
			}
		}
	}()
	return ch
}
