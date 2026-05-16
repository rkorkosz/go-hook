package transport

import "testing"

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
