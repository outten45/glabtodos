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

	"github.com/0xAX/notificator"
	simplejson "github.com/bitly/go-simplejson"
	"github.com/justincampbell/anybar"
	"github.com/namsral/flag"
)

type argsContext struct {
	Args    []string
	Host    *string
	Token   *string
	APIPath *string
	Delay   *string
	Notify  *string
	Icon    *string
}

var notify *notificator.Notificator

func (ac *argsContext) todoURL() string {
	return fmt.Sprintf("%s%stodos", *ac.Host, *ac.APIPath)
}

func (ac *argsContext) valid() bool {
	valid := true
	if *ac.Host == "" || *ac.Token == "" || *ac.APIPath == "" {
		valid = false
	}
	return valid
}

func parseArgs(args []string) *argsContext {
	fs := flag.NewFlagSetWithEnvPrefix(args[0], "GLAB", flag.ExitOnError)

	ap := &argsContext{
		Args:    args,
		Host:    fs.String("host", "", "name of the gitlab host"),
		APIPath: fs.String("apipath", "", "api path on the gitlab host"),
		Token:   fs.String("token", "", "token for gitlab"),
		Delay:   fs.String("delay", "90s", "Delay between polling gitlab. default: 90s"),
		Notify:  fs.String("notify", "", "External script to call for notifications"),
		Icon:    fs.String("icon", "", "Location of icon (optional)"),
	}
	// fmt.Printf("1ap: %+v|%+v\n", *ap.Delay, *ap.Host)
	fs.Parse(args)
	// fmt.Printf("2ap: %+v|%+v\n", *ap.Delay, *ap.Host)
	if !ap.valid() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fs.PrintDefaults()
		os.Exit(1)
	}

	return ap
}

func sendNotifications(todos []interface{}, ext_command string) {
	if len(todos) > 0 {
		t := time.Now()
		fmt.Printf("%s - TODO count found: %d\n", t.Format("2006-01-02 15:04:05"), len(todos))
		anybar.Red()
		txt := fmt.Sprintf("%d pending TODOs.", len(todos))
		err := notify.Push("GitLab Todo", txt, "", notificator.UR_NORMAL)
		if err != nil {
			log.Print("Nofificator error: ")
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

func main() {
	ac := parseArgs(os.Args)
	anybar.White()
	icon := ""
	if ac.Icon != nil && len(*ac.Icon) > 0 {
		icon = *ac.Icon
	}
	notify = notificator.New(notificator.Options{AppName: "GitLab", DefaultIcon: icon})

	// fmt.Printf("%+v\n", ac)
	var err error
	var errorCount int64

	for {
		err = checkTodos(ac)
		t, err2 := time.ParseDuration(*ac.Delay)
		// log.Printf("time: %+t\n", t)
		if err2 != nil {
			log.Fatalf("Error: %+v\n", err2)
		}

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
