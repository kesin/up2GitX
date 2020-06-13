package share

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	NetHttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/gookit/gcli/v2/interact"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

func InvalidAlert(platform string) {
	fmt.Printf("Tell me which repos dir your want to sync, Usage: ")
	color.Yellow.Printf("up2 %s /Users/Zoker/repos/\n", platform)
	fmt.Printf("See 'up2 %s -h' for more details\n", platform)
}

func DirExists(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func GetGitDir(repoDir string) (repos []string, err error) {
	repos = make([]string, 0, 10)

	dir, err := ioutil.ReadDir(repoDir)
	if err != nil {
		return nil, err
	}

	pathSep := string(os.PathSeparator)

	for _, repo := range dir {
		if !repo.IsDir() {
			continue
		}
		repoPath := repoDir + pathSep + repo.Name() // todo check repo path valid
		if isGitRepo(repoPath) {                    // todo goroutine
			repos = append(repos, repoPath)
		}
	}

	return repos, nil
}

func isGitRepo(repoPath string) (isGit bool) {
	cmd := exec.Command("git", "tag")
	cmd.Dir = repoPath
	output, _ := cmd.CombinedOutput()
	result := string(output)
	if strings.Contains(result, "not a git repository") {
		return false
	} else {
		return true
	}
}

func printRepos(repos []string) {
	color.Yellow.Println(len(repos), "repositories detected, please check below: ", "\n")
	alertFlag := false
	for i, repo := range repos { // todo goroutine
		i = i + 1
		p := fmt.Sprintf("%d. %s", i, repo)
		fmt.Printf(p)
		size, outAlert, _ := repoSize(repo)
		alertFlag = alertFlag || outAlert
		if outAlert {
			color.Red.Printf(" %.2f", size)
			color.Red.Println("M")
		} else {
			color.Green.Printf(" %.2f", size)
			color.Green.Println("M")
		}
	}

	if alertFlag {
		color.Yellow.Println("Warning: some of your local repo is out of 1G, please make sure that you account have permission to sync repository that size more than 1G")
	}
}

func repoSize(path string) (float32, bool, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})

	outOf1G := false
	if size > 1024*1024*1024 {
		outOf1G = true
	}
	sizeMB := float32(size) / 1024.0 / 1024.0
	return sizeMB, outOf1G, err
}

func ReadyToAuth(repoDir string) []string {
	if DirExists(repoDir) {
		repos, _ := GetGitDir(repoDir)
		if len(repos) == 0 {
			color.Red.Printf("No git repositories detected in %s \n", repoDir)
		} else {
			printRepos(repos)
			inPut, _ := interact.ReadLine("\nCheck if this repositories are what you expected, ready to the next step? (y/n) ")
			if inPut == "y" {
				return repos
			} else {
				ExitMessage()
			}
		}
	} else {
		color.Red.Println("The path you provided is not a dir or not exists")
	}
	return nil
}

func ExitMessage() {
	color.Yellow.Println("Bye, see you next time!")
}

func Get(url string) (map[string]interface{}, error) {
	response, err := NetHttp.Get(url)
	if err != nil {
		color.Red.Printf("Request failed, Error: %s \n", err.Error())
		return nil, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func Post(uri string, params map[string]interface{}) (map[string]interface{}, error) {
	data := url.Values{}
	for k, v := range params {
		data.Add(k, v.(string))
	}

	response, err := NetHttp.PostForm(uri, data)
	if err != nil {
		color.Red.Printf("Request failed, Error: %s \n", err.Error())
		return nil, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func PostForm(uri string, params map[string]interface{}) (map[string]interface{}, error) {
	data := ""
	for k, v := range params {
		data += fmt.Sprintf("%s=%s&%s", k, v.(string), data)
	}

	response, err := NetHttp.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(data))
	if err != nil {
		color.Red.Printf("Request failed, Error: %s \n", err.Error())
		return nil, err
	}
	defer response.Body.Close()

	body, _ := ioutil.ReadAll(response.Body)

	var result map[string]interface{}
	json.Unmarshal(body, &result)
	return result, nil
}

func ShowProjectLists(host string, repos []string, path string) {
	for i, r := range repos {
		i = i + 1
		ra := strings.Split(r, "/")
		p := fmt.Sprintf("%d. https://%s/%s/%s", i, host, path, ra[len(ra) - 1])
		color.Yellow.Println(p)
	}
}

func AskPublic(npType string) string {
	namespace := []string{"Public (Anyone can see this repository)",
						  "Private (Only members can see this repository)"}
	if npType == "Enterprise" {
		namespace = append(namespace, "Inner public (Only enterprise members can see this repository)")
	}
	fmt.Printf("\n")
	ques := "Please choose this project's public type: (new projects will apply)"
	public := selectOne(namespace, ques)
	return public
}

func AskError() string {
	howTo := []string{"Exit and fix them",
		"Skip them"}
	ques := "There are errors on some dirs, what would you like to do?"
	return selectOne(howTo, ques)
}

func AskExist() string {
	color.Notice.Println("\n", "WARNING: The exist project will remain private attribute as what it was!", "\n")
	howTo := []string{"Exit and fix them",
		"Skip them",
		"Overwrite the remote (same as git push --force, you need exactly know what you do before you select this item)"}
	ques := "The are some projects name already exists, what would you like to do?"
	return selectOne(howTo, ques)
}

func selectOne(items []string, ques string) string {
	return 	interact.SelectOne(ques, items, "",)
}

func SyncRepo(auth *http.BasicAuth ,local string, uri string, force string) error {
	var forceStr string

	// generate a tmp remote
	remote := fmt.Sprintf("up2GitX-%d", time.Now().Unix())
	r, err := git.PlainOpen(local)
	if err != nil {
		return err
	}

	// delete this remote after sync whether success or not
	defer deleteRemote(r, remote)
	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: remote,
		URLs: []string{uri},
	})
	if err != nil {
		return err
	}

	switch force { // push force or not
	case "2":
		forceStr = "+"
	default:
		forceStr = ""
	}
	rHeadStrings := fmt.Sprintf("%srefs/%s/*:refs/%s/*", forceStr, "heads", "heads")
	rTagStrings := fmt.Sprintf("%srefs/%s/*:refs/%s/*", forceStr, "tags", "tags")
	rHeads := config.RefSpec(rHeadStrings)
	rTags := config.RefSpec(rTagStrings)

	err = r.Push(&git.PushOptions{RemoteName: remote,
		RefSpecs: []config.RefSpec{rHeads, rTags},
		Auth: auth})
	if err != nil {
		return err
	}
	return nil
}

func deleteRemote(r  *git.Repository, upRe string) {
	r.DeleteRemote(upRe)
}