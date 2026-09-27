package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseMultiInstanceConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	data := `delay = "2m"
[[instances]]
name = "one"
host = "https://one.example"
api_path = "/api/v4/"
op_path = "op://one/token"
[[instances]]
name = "two"
host = "https://two.example"
api_path = "/api/v4/"
op_path = "op://two/token"
`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := parseArgs([]string{"glabtodos", "--config", path})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.instances) != 2 || cfg.instances[0].name != "one" || cfg.instances[1].opPath != "op://two/token" || cfg.delay != 2*time.Minute {
		t.Fatalf("unexpected settings: %+v", cfg)
	}
}

func TestLegacyFlags(t *testing.T) {
	cfg, err := parseArgs([]string{"glabtodos", "--no-config", "--host", "https://old.example", "--apipath", "/api/v3/", "--token", "test"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.instances) != 1 || cfg.instances[0].todoURL() != "https://old.example/api/v3/todos" || cfg.instances[0].token != "test" {
		t.Fatalf("unexpected legacy settings: %+v", cfg)
	}
}

func TestLegacyEnvironmentAndFlagPrecedence(t *testing.T) {
	t.Setenv("GLAB_HOST", "https://env.example")
	t.Setenv("GLAB_APIPATH", "/api/v4/")
	t.Setenv("GLAB_TOKEN", "env-token")
	cfg, err := parseArgs([]string{"glabtodos", "--no-config", "--host", "https://flag.example"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.instances[0].todoURL() != "https://flag.example/api/v4/todos" || cfg.instances[0].token != "env-token" {
		t.Fatalf("unexpected precedence: %+v", cfg.instances[0])
	}
}

func TestPollIndependentInstancesAndPagination(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad/todos" {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		pages = append(pages, r.URL.Query().Get("page"))
		if r.Header.Get("PRIVATE-TOKEN") != "good-token" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("incorrect request: %v", r)
		}
		if r.URL.Query().Get("page") == "1" {
			w.Header().Set("X-Next-Page", "2")
		}
		fmt.Fprint(w, `[{}]`)
	}))
	defer server.Close()
	cfg := settings{delay: time.Minute, instances: []*instance{
		{name: "good", host: server.URL, apiPath: "/good/", token: "good-token"},
		{name: "bad", host: server.URL, apiPath: "/bad/", token: "bad-token"},
	}}
	now := time.Now()
	count, unavailable, success := poll(&cfg, server.Client(), now)
	if count != 2 || !success || strings.Join(unavailable, ",") != "bad" || cfg.instances[1].failures != 1 {
		t.Fatalf("unexpected poll: %d, %v, %v", count, unavailable, success)
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Fatalf("pagination: %v", pages)
	}
	// The failed instance is in backoff; the healthy instance can poll again.
	count, unavailable, success = poll(&cfg, server.Client(), now.Add(time.Minute))
	if count != 2 || !success || len(unavailable) != 1 {
		t.Fatalf("unexpected second poll: %d, %v, %v", count, unavailable, success)
	}
	server.Close()
	count, unavailable, success = poll(&cfg, server.Client(), now.Add(2*time.Minute))
	if count != 0 || success || len(unavailable) != 2 {
		t.Fatalf("stale count included after failure: %d, %v, %v", count, unavailable, success)
	}
}

func TestRetryDelay(t *testing.T) {
	if retryDelay(1, 90*time.Second) != 90*time.Second || retryDelay(2, time.Second) != 2*time.Minute || retryDelay(100, time.Second) != 30*time.Minute {
		t.Fatal("unexpected backoff")
	}
}
