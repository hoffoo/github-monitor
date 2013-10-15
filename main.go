package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var out io.Writer
var limit int
var nodupes map[string]int
var skipEvents = []string{
	"GistEvent",
	"MemberEvent",
}

const (
	BASE_URL     = "https://api.github.com/users/%s/received_events"
	USERNAME int = 0
)

func main() {

	var count = flag.Int("m", 5, "max number of events to display")
	var debug = flag.Bool("d", false, "debug - read from file instead of request")
	flag.IntVar(&limit, "c", 0, "cut after this length of output")

	flag.Parse()
	username := flag.Arg(USERNAME)

	nodupes = make(map[string]int)

	if username == "" {
		fmt.Fprintf(os.Stderr, "Specify a username\n")
		return
	}

	var r io.Reader
	var resp *http.Response
	var err error

	if *debug == false {
		resp, err = http.Get(fmt.Sprintf(BASE_URL, username))
		r = resp.Body
		defer resp.Body.Close()
	} else {
		r, err = os.Open("sample.json")
	}

	if err != nil {
		panic(err)
	}

	result := Receive(r)

	out = os.Stdout

	skipped := 0
	for i, res := range result {
		if res.summarize() {
			skipped += 1
		}

		if i+1-skipped == *count {
			break
		}
	}
}

type GithubJSON struct {
	Actor struct {
		Login, Url string
	}
	Repo struct {
		Name, Url string
	}
	Payload struct {
		Forkee struct {
			Full_Name, Description string
		}
		Issue struct {
			Url    string
			Number int
		}
		Target struct {
			Login string
		}
		Ref      string
		Ref_Type string
		Action   string
	}
	Type string
}

func (gj *GithubJSON) summarize() (skipped bool) {

	switch gj.Type {
	case "WatchEvent":
		format("%s starred %s", gj.Actor.Login, gj.Repo.Name)
	case "FollowEvent":
		format("%s followed %s", gj.Actor.Login, gj.Payload.Target.Login)
	case "IssuesEvent":
		switch gj.Payload.Action {
		case "created":
			format("%s comment issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		case "opened":
			format("%s made issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		case "closed":
			format("%s closed issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		default:
			format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "IssueCommentEvent":
		format("%s commented %s", gj.Actor.Login, gj.Repo.Name)
	case "PushEvent":
		format("%s pushed to %s", gj.Actor.Login, gj.Repo.Name)
	case "ForkEvent":
		format("%s forked %s", gj.Actor.Login, gj.Repo.Name)
	case "CreateEvent":
		switch gj.Payload.Ref_Type {
		case "tag":
			format("%s tagged %s %s", gj.Actor.Login, gj.Payload.Ref, gj.Repo.Name)
		case "repository":
			format("%s created %s", gj.Actor.Login, gj.Repo.Name)
		default:
			format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "PullRequestReviewCommentEvent":
		format("%s commented %s", gj.Actor.Login, gj.Repo.Name)
	case "PullRequestEvent":
		switch gj.Payload.Action {
		case "closed":
			format("%s closed pull %s", gj.Actor.Login, gj.Repo.Name)
		case "opened":
			format("%s pull req %s", gj.Actor.Login, gj.Repo.Name)
		default:
			format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	default:
		for _, event := range skipEvents {
			if event == gj.Type {
				return true // we skipped this event
			}
		}
		format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
	}

	return
}

func format(f string, args ...interface{}) {

	f = fmt.Sprintf(f, args...)
	sameAuthorRepo := fmt.Sprintf("%s/", f[:strings.Index(f, " ")])
	f = strings.Replace(f, sameAuthorRepo, "", 1)

	if limit > 0 && limit < len(f) {
		f = f[:limit]
	}

	// squash duplicates
	if _,dupe := nodupes[f]; dupe == false {
		out.Write([]byte(f + "\n"))
		nodupes[f] = 0
	}
}

func Receive(r io.Reader) []GithubJSON {

	data, err := ioutil.ReadAll(r)

	if err != nil {
		panic(err)
	}

	var result []GithubJSON

	err = json.Unmarshal(data, &result)

	if err != nil {
		panic(err)
	}

	return result
}
