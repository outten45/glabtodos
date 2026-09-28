package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/justincampbell/anybar"

	"github.com/outtenr/glabtodos/secrets"
)

type instance struct {
	name, host, apiPath, opPath, opCommand, token string
	count                                         int
	known                                         bool
	failed                                        bool
	failures                                      int
	nextAttempt                                   time.Time
}

type settings struct {
	instances []*instance
	delay     time.Duration
	notify    string
	icon      string
}

func parseArgs(args []string) (settings, error) {
	file, configPath, err := loadFileConfig(args)
	if err != nil {
		return settings{}, err
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	host := fs.String("host", envOr("GLAB_HOST", file.Host), "GitLab host (single-instance mode)")
	apiPath := fs.String("apipath", envOr("GLAB_APIPATH", file.APIPath), "GitLab API path (single-instance mode)")
	token := fs.String("token", envOr("GLAB_TOKEN", ""), "GitLab token (single-instance mode; not read from config file)")
	opPath := fs.String("op-path", envOr("GLAB_OP_PATH", file.OPPath), "1Password token reference (single-instance mode)")
	opCmd := fs.String("op-command", envOr("GLAB_OP_COMMAND", defaultString(file.OPCommand, "op.exe")), "1Password CLI command")
	delay := fs.String("delay", envOr("GLAB_DELAY", defaultString(file.Delay, "90s")), "interval between polling attempts")
	notify := fs.String("notify", envOr("GLAB_NOTIFY", file.Notify), "external notification command")
	icon := fs.String("icon", envOr("GLAB_ICON", file.Icon), "notification icon")
	fs.String("config", configPath, "TOML configuration file")
	fs.Bool("no-config", false, "disable configuration file loading")
	if err := fs.Parse(args[1:]); err != nil {
		return settings{}, err
	}
	interval, err := time.ParseDuration(*delay)
	if err != nil || interval <= 0 {
		return settings{}, fmt.Errorf("delay must be a positive duration: %q", *delay)
	}
	cfg := settings{delay: interval, notify: *notify, icon: *icon}
	if len(file.Instances) > 0 {
		// Single-instance overrides are deliberately not applied to the list.
		seen := make(map[string]bool)
		for _, entry := range file.Instances {
			if entry.Name == "" || seen[entry.Name] {
				return settings{}, fmt.Errorf("instance names must be nonempty and unique: %q", entry.Name)
			}
			seen[entry.Name] = true
			i := &instance{name: entry.Name, host: entry.Host, apiPath: entry.APIPath,
				opPath: entry.OPPath, opCommand: defaultString(entry.OPCommand, *opCmd)}
			if err := i.validate(); err != nil {
				return settings{}, err
			}
			cfg.instances = append(cfg.instances, i)
		}
	} else {
		i := &instance{name: "GitLab", host: *host, apiPath: *apiPath, opPath: *opPath,
			opCommand: *opCmd, token: *token}
		if err := i.validate(); err != nil {
			return settings{}, err
		}
		cfg.instances = []*instance{i}
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (i *instance) validate() error {
	if i.host == "" || i.apiPath == "" || (i.opPath == "" && i.token == "") {
		return fmt.Errorf("instance %q requires host, api_path, and op_path or token", i.name)
	}
	u, err := url.Parse(i.host)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("instance %q has invalid host URL %q", i.name, i.host)
	}
	if !strings.HasPrefix(i.apiPath, "/") || !strings.HasSuffix(i.apiPath, "/") {
		return fmt.Errorf("instance %q: api_path must start and end with /", i.name)
	}
	return nil
}

func (i *instance) todoURL() string {
	return strings.TrimRight(i.host, "/") + i.apiPath + "todos"
}

// fetchTodos reads all pages; a failed page invalidates the entire count.
func fetchTodos(client *http.Client, i *instance) (int, error) {
	base := i.todoURL()
	count := 0
	for page := 1; ; {
		u, err := url.Parse(base)
		if err != nil {
			return 0, err
		}
		query := u.Query()
		query.Set("per_page", "100")
		query.Set("page", strconv.Itoa(page))
		u.RawQuery = query.Encode()
		req, err := http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			return 0, err
		}
		req.Header.Set("PRIVATE-TOKEN", i.token)
		resp, err := client.Do(req)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", i.name, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return 0, fmt.Errorf("%s: GitLab returned HTTP %d", i.name, resp.StatusCode)
		}
		var todos []json.RawMessage
		err = json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&todos)
		resp.Body.Close()
		if err != nil || todos == nil {
			return 0, fmt.Errorf("%s: invalid TODO response: %v", i.name, err)
		}
		count += len(todos)
		next := resp.Header.Get("X-Next-Page")
		if next == "" {
			return count, nil
		}
		nextPage, err := strconv.Atoi(next)
		if err != nil || nextPage <= page {
			return 0, fmt.Errorf("%s: invalid X-Next-Page %q", i.name, next)
		}
		page = nextPage
	}
}

// retryDelay uses independent, bounded exponential backoff for each instance.
func retryDelay(failures int, interval time.Duration) time.Duration {
	backoff := time.Minute
	for n := 1; n < failures && backoff < 30*time.Minute; n++ {
		backoff *= 2
	}
	if backoff > 30*time.Minute {
		backoff = 30 * time.Minute
	}
	if interval > backoff {
		return interval
	}
	return backoff
}

// loadTokens resolves 1Password references once at startup. Tokens remain cached
// in each instance until the process exits.
func loadTokens(cfg *settings) error {
	for _, i := range cfg.instances {
		if i.opPath == "" {
			continue
		}
		token, err := secrets.GitLabToken(i.opCommand, i.opPath)
		if err != nil {
			return fmt.Errorf("%s: unable to load 1Password token: %w", i.name, err)
		}
		i.token = token
	}
	return nil
}

// poll updates each instance independently; unavailable counts are excluded
// from the total, including counts from earlier successful polls.
func poll(cfg *settings, client *http.Client, now time.Time) (int, []string, bool) {
	var wg sync.WaitGroup
	for _, i := range cfg.instances {
		if now.Before(i.nextAttempt) {
			continue
		}
		wg.Add(1)
		go func(i *instance) {
			defer wg.Done()
			count, err := fetchTodos(client, i)
			if err != nil {
				i.failed = true
				i.failures++
				i.nextAttempt = now.Add(retryDelay(i.failures, cfg.delay))
				log.Printf("%s: %v; retrying at %s", i.name, err, i.nextAttempt.Format(time.RFC3339))
			} else {
				i.count, i.known, i.failed, i.failures = count, true, false, 0
				i.nextAttempt = now.Add(cfg.delay)
			}
		}(i)
	}
	wg.Wait()
	total := 0
	unavailable := []string{}
	anySuccess := false
	for _, i := range cfg.instances {
		if i.failed || !i.known {
			unavailable = append(unavailable, i.name)
		} else {
			total += i.count
			anySuccess = true
		}
	}
	return total, unavailable, anySuccess
}

func sendNotifications(total int, unavailable []string, anySuccess bool, cfg *settings) {
	if !anySuccess {
		log.Printf("No GitLab TODO counts available (%s)", strings.Join(unavailable, ", "))
		anybar.White()
		return
	}
	message := fmt.Sprintf("%d pending TODOs", total)
	if len(unavailable) > 0 {
		message += fmt.Sprintf(" (partial; unavailable: %s)", strings.Join(unavailable, ", "))
	}
	message += "."
	log.Println(message)
	if total == 0 {
		anybar.White()
	} else {
		anybar.Red()
	}
	if total == 0 && len(unavailable) == 0 {
		return
	}
	if err := beeep.Alert("GitLab Todo", message, cfg.icon); err != nil {
		log.Printf("Beeep notification error: %v", err)
	}
	if cfg.notify != "" {
		cmd := exec.Command(cfg.notify, message)
		if err := cmd.Run(); err != nil {
			log.Printf("External notification command error: %v", err)
		}
	}
}

func main() {
	cfg, err := parseArgs(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	if err := loadTokens(&cfg); err != nil {
		log.Fatal(err)
	}
	beeep.AppName = "GLabTodos"
	anybar.White()
	client := &http.Client{
		Timeout: 15 * time.Second,
		// Do not forward PRIVATE-TOKEN to a redirect destination.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	for {
		total, unavailable, anySuccess := poll(&cfg, client, time.Now())
		sendNotifications(total, unavailable, anySuccess, &cfg)
		time.Sleep(cfg.delay)
	}
}
