package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

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
}

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
		Delay:   fs.String("delay", "60s", "Delay between polling gitlab. default: 60s"),
	}
	fs.Parse(args)
	if !ap.valid() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fs.PrintDefaults()
		os.Exit(1)
	}

	return ap
}

func checkTodos(ac *argsContext) {

	url := ac.todoURL()

	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("PRIVATE-TOKEN", *ac.Token)

	response, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	defer response.Body.Close()
	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	j, err := simplejson.NewJson(buf)
	if err != nil {
		log.Fatal(err)
	}
	vals, err := j.Array()
	if err != nil {
		log.Fatal(err)
	}

	if len(vals) > 0 {
		t := time.Now()
		fmt.Printf("%s - Todo count found: %d\n", t.Format("2006-01-02 15:04:05"), len(vals))
		anybar.Red()
	} else {
		t := time.Now()
		fmt.Printf("%s - Nothing found.\n", t.Format("2006-01-02 15:04:05"))
		anybar.White()
	}
}

func main() {
	ac := parseArgs(os.Args)

	fmt.Printf("%+v\n", ac)

	for {
		checkTodos(ac)
		t, _ := time.ParseDuration(*ac.Delay)
		time.Sleep(t)
	}

}
