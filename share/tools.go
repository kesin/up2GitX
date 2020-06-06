package share

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gookit/color"
	"github.com/gookit/gcli/v2/interact"
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
	color.Yellow.Println(len(repos), "repositories detected, please check below: ")
	alertFlag := false
	for _, repo := range repos { // todo goroutine
		fmt.Printf(repo)
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
			inPut, _ := interact.ReadLine("Check if this repositories are what you expected, ready to the next step? (y/n) ")
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
	response, err := http.Get(url)
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

	response, err := http.PostForm(uri, data)
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
