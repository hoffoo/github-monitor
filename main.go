package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var out io.Writer
var limit int
var nodupes map[string]int // TODO make this an array
var showDupes bool
var debugOut io.Writer
var debug bool

var skipEvents = []string{
	"GistEvent",
	"MemberEvent",
}

const (
	DEBUG_FILE = "/tmp/github-monitor.json"
	USER_URL   = "https://api.github.com/users/%s/received_events"
	SEARCH_URL = "https://api.github.com/search/repositories?q=language:%s&sort=stars"
)

func main() {

	var username string
	var search string
	var count = flag.Int("m", 0, "max number of items to display")
	flag.IntVar(&limit, "c", 0, "cut text after this length of output")
	flag.BoolVar(&showDupes, "d", false, "show duplicate events")
	flag.StringVar(&username, "u", "", "username, get recent events")
	flag.StringVar(&search, "l", "", "language, get the top projects created this month")
	flag.BoolVar(&debug, "debug", false, "write github response to "+DEBUG_FILE)
	flag.Parse()

	nodupes = make(map[string]int)
	out = os.Stdout

	var dest string
	if search != "" && username == "" {
		now := time.Now()
		localtime, _ := time.LoadLocation("Local") // may be buggy, should do somethign with err
		minus10 := time.Date(now.Year(), now.Month(), now.Day()-10, 0, 0, 0, 0, localtime)
		sarg := fmt.Sprintf("%s created:>%s-%s-%s",
			search,
			strconv.Itoa(minus10.Year()),
			strconv.Itoa(int(minus10.Month())),
			strconv.Itoa(minus10.Day()))
		dest = fmt.Sprintf(SEARCH_URL, url.QueryEscape(sarg))
	} else if username != "" && search == "" {
		dest = fmt.Sprintf(USER_URL, url.QueryEscape(username))
	} else {
		fmt.Fprintln(os.Stderr, "pass either -l or -s, not both")
		return
	}

	// setup debug file
	if debug == true {
		debugOut = os.Stdout
	}

	result := receive(dest)

	// TODO move this outside of main
	skipped := 0
	for i, res := range *result {
		if res.summarize() {
			skipped += 1
		}

		if i+1-skipped == *count {
			break
		}
	}
}

func receive(dest string) *[]GithubJSON {

	header := &http.Header{}
	header.Add("Accept", "application/vnd.github.preview")

	req, _ := http.NewRequest("GET", dest, nil)
	req.Header = *header
	client := &http.Client{}
	resp, _ := client.Do(req)

	data, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if debug == true {
		var debugBuff bytes.Buffer
		json.Indent(&debugBuff, data, "", "\t")
		debugOut.Write(debugBuff.Bytes())
		debugOut.Write([]byte("\n\n"))
	}

	var result []GithubJSON
	err := json.Unmarshal(data, &result)

	// if fail try to unmarshall as search result
	if err != nil {
		searchHolder := struct {
			Items *[]GithubJSON
		}{&result}

		err := json.Unmarshal(data, &searchHolder)

		if err != nil {
			panic("couldnt unmarshal github response")
		}
	}

	return &result
}

/*
holds both user stats data and search response json */
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
		Release struct {
			Tag_Name string
		}
		Ref      string
		Ref_Type string
		Action   string
	}
	Type           string
	Full_Name      string
	Watchers_Count int
	Forks_Count    int
}

func (gj *GithubJSON) GetType() string {
	if gj.Type == "" {
		return "Project"
	}
	return gj.Type
}

func (gj *GithubJSON) summarize() (skipped bool) {

	switch gj.GetType() {
	case "Project":
		skipped = format("%s %s", gj.Full_Name, strconv.Itoa(gj.Watchers_Count))
	case "WatchEvent":
		skipped = format("%s star %s", gj.Actor.Login, gj.Repo.Name)
	case "FollowEvent":
		skipped = format("%s follow %s", gj.Actor.Login, gj.Payload.Target.Login)
	case "IssuesEvent":
		switch gj.Payload.Action {
		case "created":
			skipped = format("%s comment issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		case "opened":
			skipped = format("%s made issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		case "closed":
			skipped = format("%s close issue %d %s", gj.Actor.Login, gj.Payload.Issue.Number, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "IssueCommentEvent":
		skipped = format("%s comment issue %s", gj.Actor.Login, gj.Repo.Name)
	case "PushEvent":
		skipped = format("%s push to %s", gj.Actor.Login, gj.Repo.Name)
	case "ForkEvent":
		skipped = format("%s fork %s", gj.Actor.Login, gj.Repo.Name)
	case "CreateEvent":
		switch gj.Payload.Ref_Type {
		case "tag":
			skipped = format("%s tag %s %s", gj.Actor.Login, gj.Payload.Ref, gj.Repo.Name)
		case "repository":
			skipped = format("%s create %s", gj.Actor.Login, gj.Repo.Name)
		case "branch":
			skipped = format("%s branch %s", gj.Actor.Login, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "PullRequestReviewCommentEvent":
		skipped = format("%s comment req %s", gj.Actor.Login, gj.Repo.Name)
	case "PullRequestEvent":
		switch gj.Payload.Action {
		case "closed":
			skipped = format("%s close pull %s", gj.Actor.Login, gj.Repo.Name)
		case "opened":
			skipped = format("%s create pull %s", gj.Actor.Login, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "DeleteEvent":
		switch gj.Payload.Ref_Type {
		case "branch":
			skipped = format("%s del branch %s %s", gj.Actor.Login, gj.Payload.Ref, gj.Repo.Name)
		default:
			skipped = format("-> %s %s %s", gj.Type, gj.Actor.Login, gj.Repo.Name)
		}
	case "ReleaseEvent":
		switch gj.Payload.Action {
		case "published":
			skipped = format("%s published %s %s", gj.Actor.Login, gj.Payload.Release.Tag_Name, gj.Repo.Name)
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
	idx := strings.Index(f, " ")
	if idx > -1 {
		sameAuthorRepo := fmt.Sprintf("%s/", f[:strings.Index(f, " ")])
		f = strings.Replace(f, sameAuthorRepo, "", 1)
	}

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
