package notify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWebhookPostsJSON(t *testing.T) {
	got := make(chan Event, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %q", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		var e Event
		_ = json.Unmarshal(body, &e)
		got <- e
	}))
	defer srv.Close()

	Webhook{URL: srv.URL}.Notify(Event{Title: "hi", Body: "there"})

	select {
	case e := <-got:
		if e.Title != "hi" || e.Body != "there" {
			t.Fatalf("received %+v", e)
		}
		if e.Time == "" {
			t.Error("Time should be filled in")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook was not delivered")
	}
}

func TestWebhookEmptyURLNoop(t *testing.T) {
	// Must not panic or block.
	Webhook{}.Notify(Event{Title: "x"})
}

func TestMultiFansOut(t *testing.T) {
	c := make(chan Event, 2)
	rec := recorder{c}
	Multi{rec, rec}.Notify(Event{Title: "t"})
	for i := 0; i < 2; i++ {
		select {
		case <-c:
		case <-time.After(time.Second):
			t.Fatal("missing fan-out delivery")
		}
	}
}

type recorder struct{ c chan Event }

func (r recorder) Notify(e Event) { r.c <- e }
