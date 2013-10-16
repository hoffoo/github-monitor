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
var showDupes bool

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
	flag.IntVar(&limit, "c", 0, "cut after this length of output")
	flag.BoolVar(&showDupes, "d", false, "show duplicate events")

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

	resp, err = http.Get(fmt.Sprintf(BASE_URL, username))
	r = resp.Body
	defer resp.Body.Close()

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
		skipped = format("%s starred %s", gj.Actor.Login, gj.Repo.Name)
	case "FollowEvent":
		skipped = format("%s followed %s", gj.Actor.Login, gj.Payload.Target.Login)
	case "IssuesEvent":
		switch gj.Payload.Action {
		case "created":
			skipped = format("%s comment issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		case "opened":
			skipped = format("%s made issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		case "closed":
			skipped = format("%s closed issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "IssueCommentEvent":
		skipped = format("%s commented %s", gj.Actor.Login, gj.Repo.Name)
	case "PushEvent":
		skipped = format("%s pushed to %s", gj.Actor.Login, gj.Repo.Name)
	case "ForkEvent":
		skipped = format("%s forked %s", gj.Actor.Login, gj.Repo.Name)
	case "CreateEvent":
		switch gj.Payload.Ref_Type {
		case "tag":
			skipped = format("%s tagged %s %s", gj.Actor.Login, gj.Payload.Ref, gj.Repo.Name)
		case "repository":
			skipped = format("%s created %s", gj.Actor.Login, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "PullRequestReviewCommentEvent":
		skipped = format("%s commented %s", gj.Actor.Login, gj.Repo.Name)
	case "PullRequestEvent":
		switch gj.Payload.Action {
		case "closed":
			skipped = format("%s closed pull %s", gj.Actor.Login, gj.Repo.Name)
		case "opened":
			skipped = format("%s pull req %s", gj.Actor.Login, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	default:
		for _, event := range skipEvents {
			if event == gj.Type {
				return true // we skipped this event
			}
		}
		skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
	}

	return skipped
}

func format(f string, args ...interface{}) (skipped bool) {

	f = fmt.Sprintf(f, args...)
	sameAuthorRepo := fmt.Sprintf("%s/", f[:strings.Index(f, " ")])
	f = strings.Replace(f, sameAuthorRepo, "", 1)

	if limit > 0 && limit < len(f) {
		f = f[:limit]
	}

	if showDupes == true {
		out.Write([]byte(f + "\n"))
		return false // always show dupes, dont mark skipped
	}

	// squash duplicates
	if _, dupe := nodupes[f]; dupe == false {
		out.Write([]byte(f + "\n"))
		nodupes[f] = 0
	} else {
		return true // skipped because dupe
	}

	return false
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
