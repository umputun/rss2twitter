package rss

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotify(t *testing.T) {
	var n int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fnum := atomic.AddInt32(&n, int32(1))
		if fnum > 2 {
			fnum = 2
		}
		data, err := ioutil.ReadFile(fmt.Sprintf("testdata/f%d.xml", fnum))
		require.NoError(t, err)
		w.WriteHeader(200)
		w.Write(data)
	}))

	defer ts.Close()
	notify := Notify{Feed: ts.URL, Duration: time.Millisecond * 250, Timeout: time.Millisecond * 100}
	ch := notify.Go(context.Background())
	defer notify.Shutdown()

	st := time.Now()
	e := <-ch
	t.Logf("%+v", e)
	e.Text = ""
	assert.Equal(t, Event{ChanTitle: "Радио-Т", Title: "Радио-Т 626",
		Link: "https://radio-t.com/p/2018/12/01/podcast-626/", GUID: "https://radio-t.com/p/2018/12/01//podcast-626/"}, e)
	assert.True(t, time.Since(st) >= time.Millisecond*250)

	select {
	case <-ch:
		t.Fatal("should not get any more")
	default:
	}
}
