package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
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
		data, err := ioutil.ReadFile(fmt.Sprintf("rss/testdata/f%d.xml", fnum))
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

func TestExclusionPatterns(t *testing.T) {
	excludes := []string {
		"^The",
		"$end",
		"^The end$",
		"roar",
	}
	tbl := []struct {
		msg  		string
		result  	bool
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
