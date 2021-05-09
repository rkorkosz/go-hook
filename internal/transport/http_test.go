package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

func TestHTTPTransport(t *testing.T) {
	ht := NewHTTP()
	ts := httptest.NewServer(ht)
	defer ts.Close()
	request, err := http.NewRequest("GET", ts.URL+"/test/user", nil)
	if err != nil {
		t.Error(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	request.WithContext(ctx)
	go func() {
		_, err = ts.Client().Do(request)
		if err != nil {
			t.Error(err)
		}
	}()
	var msg bytes.Buffer
	_, err = msg.WriteString(`{"a":1}`)
	if err != nil {
		t.Error(err)
	}
	res, err := ts.Client().Post(ts.URL+"/test/user", "application/json", &msg)
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 202 {
		t.Errorf("wrong response code: %d", res.StatusCode)
	}
	cancel()
	var data pubsub.Data
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		t.Error(err)
	}
	if bytes.Equal(data.Data, msg.Bytes()) {
		t.Errorf("Want: %s, got: %s", msg.String(), data.Data)
	}
}
