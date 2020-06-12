package platform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"up2GitX/share"

	"github.com/gookit/gcli/v2"
	"github.com/gookit/color"
	"github.com/gookit/gcli/v2/interact"
	"github.com/gookit/gcli/v2/progress"
)

type RepoResult struct {
	local  string
	uri  string
	status  int
	error string
}

const (
	SUCCESS int = 0
	EXIST int   = 1
	ERROR int   = 2
	SPL string = "------------------------------"
)

func GiteeCommand() *gcli.Command {
	gitee := &gcli.Command{
		Func:     syncGitee,
		Name:     "gitee",
		UseFor:   "This command is used for sync local repo to Gitee",
		Examples: `Simple usage: <cyan>{$binName} {$cmd} /Users/Zoker/repos/</>`,
	}

	// bind args with names
	gitee.AddArg("repoDir", "Tell me which repos your want to sync, is required", false)

	return gitee
}

func syncGitee(c *gcli.Command, args []string) error {
	// check message
	if len(args) == 0 {
		share.InvalidAlert(c.Name)
		return nil
	}

	// check repodir and print projects to ensure
	repos := share.ReadyToAuth(args[0])
	if repos == nil {
		return nil
	}

	// enter userinfo to get access token
	askResult, success := askForAccount()
	if !success {
		color.Red.Println(askResult)
		return nil
	}
	accessToken := askResult

	// get userinfo via access token
	userInfo, success := getUserInfo(accessToken)
	if !success {
		color.Red.Println(userInfo["error"])
		return nil
	}
	color.Green.Printf("Hello, %s! \n", userInfo["name"])

	// get available namespace todo: enterprise and group
	allNamespace := getNamespace(userInfo)
	namespace := make([]string, len(allNamespace))
	for i, n := range allNamespace {
		namespace[i] = n[1]
	}
	selectedNumber := askNamespace(namespace)
	numberD, _ := strconv.Atoi(selectedNumber)

	// select namespace and ask for ensure
	selectedNp := allNamespace[numberD]
	fmt.Printf("Selected %s(https://gitee.com/%s) as namespace, Type: %s \n" +
		"The following projects will be generated on Gitee: \n", selectedNp[0], selectedNp[1], selectedNp[2])

	// show projects list and ensure
	share.ShowProjectLists("gitee.com", repos, selectedNp[1])

	// ask for public or not
	public := share.AskPublic(selectedNp[2])

	// create projects
	color.Green.Println("Creating Projects, Please Wait...")
	repoRes := createProjects(repos, public, accessToken, selectedNp)

	// show results
	showRepoRes(repoRes)

	// ask for exist and error details
	asErr := share.AskError()
	if asErr == "0" {
		return nil
	}
	asExi := share.AskExist()
	if asExi == "0" {
		return nil
	}
	fmt.Println(asExi)
	return nil
}

func askForAccount() (string, bool) {
	email, _ := interact.ReadLine("Please enter your Gitee email: ")
	password := interact.ReadPassword("Please enter your Gitee password: ")
	if len(email) == 0 || len(password) == 0 {
		return "Email or Password must be provided!", false
	} else {
		params := fmt.Sprintf(`{
					"grant_type": "password",
					"username": "%s",
					"password": "%s",
					"client_id": "xxxx",
					"client_secret": "xxxx",
					"scope": "user_info projects groups enterprises"
					}`, email, password)

		var paramsJson map[string]interface{}
		json.Unmarshal([]byte(params), &paramsJson)
		result, err := share.Post("https://gitee.com/oauth/token", paramsJson)

		if err != nil {
			return err.Error(), false
		}

		filterVal, ok := filterResult(result, "access_token")
		return filterVal, ok
	}
}

func getUserInfo(token string) (map [string]string, bool) {
	info := make(map [string]string)
	uri := fmt.Sprintf("https://gitee.com/api/v5/user?access_token=%s", token)
	result, err := share.Get(uri)
	if err != nil {
		info["error"] = err.Error()
		return info, false
	}
	name, ok := filterResult(result, "name")
	if !ok {
		info["error"] = name
		return info, ok
	}
	info["name"] = name
	username, ok := filterResult(result, "login")
	info["username"] = username
	return info, ok
}

func filterResult(result map[string]interface{}, key string) (string, bool) {
	val, atok := result[key].(string)
	_, errok := result["error"].(string)
	if atok {
		return val, true
	} else if errok {
		return result["error_description"].(string), false
	}
	return "Unexpectedly exit", false
}

// todo enable select group and enterprise
func getNamespace(userInfo map [string]string) [][]string {
	namespace := make([][]string, 1)
	namespace[0] = make([]string, 4)
	namespace[0][0] = userInfo["name"]
	namespace[0][1] = userInfo["username"]
	namespace[0][2] = "Personal"
	namespace[0][3] = "0"
	return namespace
}

func askNamespace(namespace []string) string {
	np := interact.SelectOne(
		"Please select which namespace you want to put this repositories: ",
		namespace,
		"",
	)
	return np
}

func createProjects(repos []string, public string, token string, np []string) []RepoResult {
	repoUrl := getRepoUrl(np)
	repoRes := make([]RepoResult, len(repos))
	step := progress.Bar(len(repos))
	step.Start()
	for i, repo := range repos { // todo: goroutine
		paths := strings.Split(repo, "/")
		path := paths[len(paths) - 1]
		params := fmt.Sprintf(`{
					"access_token": "%s",
					"name": "%s",
					"path": "%s",
					"private": "%s"
					}`, token, path, path, public)
		var paramsJson map[string]interface{}
		json.Unmarshal([]byte(params), &paramsJson)
		result, err := share.PostForm(repoUrl, paramsJson)
		if err != nil {
			return []RepoResult{}
		}
		uri, eType := filterProjectResult(result, "html_url")
		errMsg := uri
		if eType == EXIST {
			uri = fmt.Sprintf("https://gitee.com/%s/%s.git", np[1], path)
		}
		repoRes[i] = RepoResult{local: repo, uri: uri, status: eType, error: errMsg}
		i = i + 1
		step.Advance()
	}
	step.Finish()
	return repoRes
}

func getRepoUrl(np []string) string {
	var uri string
	switch np[2] {
	case "Personal":
		uri = "https://gitee.com/api/v5/user/repos"
	case "Group":
		uri = fmt.Sprintf("https://gitee.com/api/v5/orgs/%s/repos", np[1])
	case "Enterprise":
		uri = fmt.Sprintf("https://gitee.com/api/v5/enterprises/%s/repos", np[1])
	}
	return uri
}

func filterProjectResult(result map[string]interface{}, key string) (string, int) {
	var err string
	var eType int
	if result["error"] != nil {
		for k, v := range result["error"].(map[string]interface{}) {
			err = fmt.Sprint(v) // skip Type Assertion
			if k == "base" {
				eType = EXIST
			} else {
				eType = ERROR
			}
		}
		return err, eType
	}
	val, atok := result[key].(string)
	if atok {
		return val, SUCCESS
	}
	return "Unexpectedly exit", ERROR
}

func showRepoRes(repoRes []RepoResult) {
	printRepo(repoRes, SUCCESS)
	printRepo(repoRes, EXIST)
	printRepo(repoRes, ERROR)
}

func printRepo(repoRes []RepoResult, status int) {
	var p, result string
	repoStatus := repoStatus(status)
	for _, item := range repoRes {
		if item.status == status {
			if status == ERROR {
				result = item.error
			} else {
				result = item.uri
			}
			p = fmt.Sprintf("Dir: (%s)\n Status: %s\n Result: %s\n%s", item.local, repoStatus, result, SPL)
			colorRepo(status, p)
		}
	}
}

func repoStatus(status int) string {
	str := ""
	switch status {
	case SUCCESS:
		str = "Created"
	case EXIST:
		str = "Exists"
	case ERROR:
		str = "Error"
	default:
		str = "Unknown Error"
	}
	return str
}

func colorRepo(status int, p string) {
	switch status {
	case SUCCESS:
		color.Blue.Println(p)
	case EXIST:
		color.Yellow.Println(p)
	case ERROR:
		color.Red.Println(p)
	default:
		color.Red.Println(p)
	}
}