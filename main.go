package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"time"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/gen2brain/beeep"
	"github.com/justincampbell/anybar"
	"github.com/namsral/flag"

	"github.com/outtenr/glabtodos/secrets"
)

type argsContext struct {
	Args    []string
	Host    *string
	Token   *string
	OPPath  *string
	OPCmd   *string
	APIPath *string
	Delay   *string
	Notify  *string
	Icon    *string
	Config  *string
}

func (ac *argsContext) todoURL() string {
	return fmt.Sprintf("%s%stodos", *ac.Host, *ac.APIPath)
}

func (ac *argsContext) valid() bool {
	valid := true
	if *ac.Host == "" || *ac.APIPath == "" || (*ac.Token == "" && *ac.OPPath == "") {
		valid = false
	}
	return valid
}

func parseArgs(args []string) *argsContext {
	file, configPath, err := loadFileConfig(args)
	if err != nil {
		log.Fatal(err)
	}
	if file.OPCommand == "" {
		file.OPCommand = "op.exe"
	}
	fs := flag.NewFlagSetWithEnvPrefix(args[0], "GLAB", flag.ExitOnError)

	ap := &argsContext{
		Args:    args,
		Host:    fs.String("host", file.Host, "name of the gitlab host"),
		APIPath: fs.String("apipath", file.APIPath, "api path on the gitlab host"),
		Token:   fs.String("token", "", "token for gitlab (not read from the config file)"),
		OPPath:  fs.String("op-path", file.OPPath, "1Password secret reference for the GitLab token (for example, op://Personal/GitLab/API Token)"),
		OPCmd:   fs.String("op-command", file.OPCommand, "1Password CLI command"),
		Delay:   fs.String("delay", defaultString(file.Delay, "90s"), "Delay between polling gitlab. default: 90s"),
		Notify:  fs.String("notify", file.Notify, "External script to call for notifications"),
		Icon:    fs.String("icon", file.Icon, "Location of icon (optional)"),
		Config:  fs.String("config", configPath, "Path to TOML configuration file"),
	}
	fs.Bool("no-config", false, "Disable configuration file loading")
	fs.Parse(args)
	// fmt.Printf("2ap: %+v|%+v\n", *ap.Delay, *ap.Host)
	if !ap.valid() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fs.PrintDefaults()
		os.Exit(1)
	}

	return ap
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// tokenFromArgs returns the configured token. When an 1Password path is set,
// it keeps retrying until the 1Password CLI is available and authenticated.
func tokenFromArgs(ac *argsContext) string {
	if *ac.OPPath == "" {
		return *ac.Token
	}

	for {
		token, err := secrets.GitLabToken(*ac.OPCmd, *ac.OPPath)
		if err == nil {
			return token
		}
		log.Printf("Unable to read GitLab token from 1Password: %v; retrying in 5s", err)
		time.Sleep(5 * time.Second)
	}
}

func sendNotifications(todos []interface{}, ext_command string) {
	if len(todos) > 0 {
		t := time.Now()
		fmt.Printf("%s - TODO count found: %d\n", t.Format("2006-01-02 15:04:05"), len(todos))
		anybar.Red()
		txt := fmt.Sprintf("%d pending TODOs.", len(todos))
		err := beeep.Alert("GitLab Todo", txt, notificationIcon)
		if err != nil {
			log.Print("Beeep notification error: ")
			log.Println(err)
		}
		if ext_command != "" {
			cmd := exec.Command(ext_command, txt)
			err2 := cmd.Start()
			if err2 != nil {
				log.Fatal(err2)
			}
			err2 = cmd.Wait()
			if err2 != nil {
				log.Printf("External command finished with error: %v", err2)
			}
		}
	} else {
		t := time.Now()
		fmt.Printf("%s - Nothing found.\n", t.Format("2006-01-02 15:04:05"))
		anybar.White()
	}
}

func checkTodos(ac *argsContext) error {

	url := ac.todoURL()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("PRIVATE-TOKEN", *ac.Token)

	response, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return err
	}

	defer response.Body.Close()
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return err
	}

	j, err := simplejson.NewJson(buf)
	if err != nil {
		log.Println(err)
		return err
	}
	//fmt.Printf("%+v\n", j)
	vals, err := j.Array()
	if err != nil {
		log.Println(err)
		return err
	}
	sendNotifications(vals, *ac.Notify)
	return nil
}

var notificationIcon string

func main() {
	ac := parseArgs(os.Args)
	anybar.White()
	if ac.Icon != nil {
		notificationIcon = *ac.Icon
	}
	beeep.AppName = "GLabTodos"

	// Resolve the token once at startup. This also lets the application start
	// before 1Password has finished launching or signing in.
	*ac.Token = tokenFromArgs(ac)

	// fmt.Printf("%+v\n", ac)
	var err error
	var errorCount int64
	t, err2 := time.ParseDuration(*ac.Delay)
	// log.Printf("time: %+t\n", t)
	if err2 != nil {
		log.Fatalf("Error: %+v\n", err2)
	}

	for {
		err = checkTodos(ac)

		if err != nil {
			errorCount = errorCount + 1
			backoff := math.Exp2(float64(errorCount)) - 1
			fmt.Printf(">> There was a problem. Waiting %0.1f min to retry request.\n", backoff)
			t = time.Duration(backoff) * time.Minute
		} else {
			errorCount = 0
		}

		time.Sleep(t)
	}

}
