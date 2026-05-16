package transport

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublishToServer(t *testing.T) {
	t.Parallel()

	body := `{"hello":"world"}`
	requestSeen := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		if r.Method != http.MethodPost {
			t.Errorf("want method %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.Path != "/topic/source" {
			t.Errorf("want path /topic/source, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("want content type application/json, got %s", got)
		}
		if got := r.Header.Get("Origin"); got == "" {
			t.Error("expected origin header")
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if string(data) != body {
			t.Errorf("want body %s, got %s", body, string(data))
		}
		requestSeen <- struct{}{}
	}))
	defer server.Close()

	ht := &HTTP{Log: log.New(io.Discard, "", 0)}
	ht.publishToServer(server.URL, "source", "topic", []byte(body))

	select {
	case <-requestSeen:
	default:
		t.Fatal("server did not receive publish")
	}
}

func TestPublishToServerInvalidURLDoesNotPanic(t *testing.T) {
	t.Parallel()

	ht := &HTTP{Log: log.New(io.Discard, "", 0)}
	ht.publishToServer("://bad-url", "source", "topic", []byte(`{"hello":"world"}`))
}

func TestSubscribePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "topic", path: "/topic", want: "topic"},
		{name: "topic with trailing slash", path: "/topic/", want: "topic"},
		{name: "empty", path: "/", wantErr: true},
		{name: "missing", path: "", wantErr: true},
		{name: "too many parts", path: "/topic/source", wantErr: true},
		{name: "empty part", path: "/topic//source", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, err := subscribePath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if topic != tt.want {
				t.Fatalf("want %q, got %q", tt.want, topic)
			}
		})
	}
}

func TestPublishPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		wantTopic  string
		wantSource string
		wantErr    bool
	}{
		{name: "topic and source", path: "/topic/source", wantTopic: "topic", wantSource: "source"},
		{name: "trailing slash", path: "/topic/source/", wantTopic: "topic", wantSource: "source"},
		{name: "empty", path: "/", wantErr: true},
		{name: "missing source", path: "/topic", wantErr: true},
		{name: "too many parts", path: "/topic/source/extra", wantErr: true},
		{name: "empty part", path: "/topic//source", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, source, err := publishPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if topic != tt.wantTopic || source != tt.wantSource {
				t.Fatalf("want %q/%q, got %q/%q", tt.wantTopic, tt.wantSource, topic, source)
			}
		})
	}
}
