package platform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"up2GitX/share"

	"github.com/gookit/gcli/v2"
	"github.com/gookit/color"
	"github.com/gookit/gcli/v2/interact"
	"github.com/gookit/gcli/v2/progress"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
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
	SKIP string = "1"
	SPL string = "\n"
	WORKER = 5
)

func GiteeCommand() *gcli.Command {
	gitee := &gcli.Command{
		Func:     syncGitee,
		Name:     "gitee",
		UseFor:   "This command is used for sync local repo to Gitee",
		Examples: `
  <yellow>Using dir: </> <cyan>{$binName} {$cmd} /Zoker/repos/</>
  Dir example
	<gray>$ ls -l /Zoker/repos/</>
	drwxr-xr-x  4 zoker  128B Jun  1 19:05 git-work-repo1
	drwxr-xr-x  4 zoker  128B Jun  1 19:02 taskover
	drwxr-xr-x  4 zoker  128B Jun  1 19:03 blogine
	drwxr-xr-x  3 zoker   96B Jun  1 12:15 git-bare-repo3
	...

  <yellow>Using file: </> <cyan>{$binName} {$cmd} /Zoker/repos.list</>
  File example
	<gray>$ cat /Zoker/repos.list</>
	/tmp/repos/git-work-repo1
	/Zoker/workspace/git-work-repo2
	/other/path/to/git-bare-repo3
	...`}

	// bind args with names
	gitee.AddArg("repoSource", "Tell me which repo dir or list your want to sync, is required", false)

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
	askResult, success, auth := askForAccount()
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
	color.Green.Printf("\nHello, %s! \n\n", userInfo["name"])

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
	color.Notice.Printf("\nSelected %s(https://gitee.com/%s) as namespace, Type: %s \n" +
		"The following projects will be generated on Gitee: \n\n", selectedNp[0], selectedNp[1], selectedNp[2])

	// show projects list and ensure
	share.ShowProjectLists("gitee.com", repos, selectedNp[1])

	// ask for public or not
	public := share.AskPublic(selectedNp[2])

	// create projects
	fmt.Println("\n", "Creating Projects, Please Wait...")
	repoRes := generateProjects(repos, public, accessToken, selectedNp)

	// show results
	_, exiNum, errNum := showRepoRes(repoRes)

	if errNum == len(repoRes) {
		color.Red.Println("No repositories are available to be uploaded!")
		return nil
	}
	if errNum > 0 {
		asErr := share.AskError()
		if asErr == "0" {
			return nil
		}
	}
	var asExi string
	if exiNum > 0 {
		asExi = share.AskExist()
		if asExi == "0" {
			return nil
		}
	}

	// available check
	avaiRepo := availableRepo(repoRes, asExi)
	if len(avaiRepo) == 0 {
		color.Red.Println("No repositories are available to be uploaded!")
		return nil
	}

	// sync code
	fmt.Println("\n", "Syncing Projects to Gitee, Please Wait...")
	syncRes := multiSync(avaiRepo, auth, asExi)
	showSyncRes(syncRes)
	return nil
}

func askForAccount() (string, bool, *http.BasicAuth) {
	email, _ := interact.ReadLine("\nPlease enter your Gitee email: ")
	password := interact.ReadPassword("Please enter your Gitee password: ")
	if len(email) == 0 || len(password) == 0 {
		return "Email or Password must be provided!", false, nil
	} else {
		params := fmt.Sprintf(`{
					"grant_type": "password",
					"username": "%s",
					"password": "%s",
					"client_id": "xxxx", // client id from Gitee
					"client_secret": "xxxx", // client secret from Gitee
					"scope": "user_info projects groups enterprises"
					}`, email, password)

		var paramsJson map[string]interface{}
		json.Unmarshal([]byte(params), &paramsJson)
		result, err := share.Post("https://gitee.com/oauth/token", paramsJson)

		if err != nil {
			return err.Error(), false, nil
		}

		filterVal, ok := filterResult(result, "access_token")
		auth := &http.BasicAuth{email, password}
		return filterVal, ok, auth
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

func generateProjects(repos []string, public string, token string, np []string) (repoRes []RepoResult) {
	step := progress.Bar(len(repos))
	var wg sync.WaitGroup
	var mutex = &sync.Mutex{}
	paths := make(chan string)

	step.Start()
	wg.Add(len(repos))

	for w := 1; w <= WORKER; w++ {
		go createProjectWorker(paths, public, token, np, &wg, &repoRes, mutex, step)
	}

	for _, p := range repos {
		paths <- p
	}
	close(paths)

	wg.Wait()
	step.Finish()

	fmt.Printf(SPL)
	return repoRes
}

func createProjectWorker(paths chan string, public string, token string, np []string, wg *sync.WaitGroup, repoRes *[]RepoResult, mutex *sync.Mutex, step *progress.Progress) {
	for path := range paths {
		createProject(path, public, token, np, repoRes, wg, mutex)
		step.Advance()
		wg.Done()
	}
}

func createProject(path string, public string, token string, np []string, repoRes *[]RepoResult, wg *sync.WaitGroup, mutex *sync.Mutex) {
	repoUrl := getRepoUrl(np)
	phs := strings.Split(path, "/")
	repoPath := phs[len(phs) - 1]
	params := fmt.Sprintf(`{
					"access_token": "%s",
					"name": "%s",
					"path": "%s",
					"private": "%s"
					}`, token, repoPath, repoPath, public)
	var paramsJson map[string]interface{}
	json.Unmarshal([]byte(params), &paramsJson)
	result, err := share.PostForm(repoUrl, paramsJson)
	if err != nil {
		return
	}
	uri, eType := filterProjectResult(result, "html_url")
	errMsg := uri
	if eType == EXIST {
		uri = fmt.Sprintf("https://gitee.com/%s/%s.git", np[1], repoPath)
	}
	mutex.Lock()
	*repoRes = append(*repoRes, RepoResult{local: path, uri: uri, status: eType, error: errMsg})
	mutex.Unlock()
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

func showRepoRes(repoRes []RepoResult) (int, int, int) {
	success := printRepo(repoRes, SUCCESS)
	exist := printRepo(repoRes, EXIST)
	errNum := printRepo(repoRes, ERROR)
	return success, exist, errNum
}

func printRepo(repoRes []RepoResult, status int) int {
	var p, result string
	num := 0
	repoStatus := repoStatus(status)
	for _, item := range repoRes {
		if item.status == status {
			num = num + 1
			if status == ERROR {
				result = item.error
			} else {
				result = item.uri
			}
			p = fmt.Sprintf("Dir: (%s)\n  Status: %s\n  Result: ", item.local, repoStatus)
			colorRepo(status, p)
			colorResult(status, result)
			fmt.Printf(SPL)
		}
	}
	return num
}

func showSyncRes(syncRes []RepoResult) {
	printSync(syncRes, SUCCESS)
	printSync(syncRes, ERROR)
}

func printSync(syncRes []RepoResult, status int) {
	var p, result string
	for _, item := range syncRes {
		if item.status == status {
			if status == SUCCESS {
				result = "Sync to Gitee SUCCESS!"
			} else {
				result = item.error
			}
			p = fmt.Sprintf("Dir: (%s)\n  Gitee: %s\n  Result: ", item.local, item.uri)
			colorRepo(EXIST, p)
			colorResult(item.status, result)
			fmt.Printf(SPL)
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
		color.Green.Printf(p)
	case EXIST:
		color.Yellow.Printf(p)
	case ERROR:
		color.Red.Printf(p)
	default:
		color.Red.Printf(p)
	}
}

func colorResult(status int, p string) {
	switch status {
	case SUCCESS:
		color.Green.Println(p)
	case EXIST:
		color.Yellow.Println(p)
	case ERROR:
		color.Red.Println(p)
	default:
		color.Red.Println(p)
	}
}

func multiSync(avaiRepo []RepoResult, auth *http.BasicAuth, force string) (syncRes []RepoResult) {
	step := progress.Bar(len(avaiRepo))
	var wg sync.WaitGroup
	var mutex = &sync.Mutex{}
	avais := make(chan RepoResult)

	step.Start()
	wg.Add(len(avaiRepo))

	for w := 1; w <= WORKER; w++ {
		go multiSyncWorker(avais, auth, force, &syncRes, &wg, mutex, step)
	}

	for _, p := range avaiRepo {
		avais <- p
	}
	close(avais)

	wg.Wait()
	step.Finish()

	fmt.Printf(SPL)
	return syncRes
}

func multiSyncWorker(avais chan RepoResult, auth *http.BasicAuth, force string, syncRes *[]RepoResult, wg *sync.WaitGroup, mutex *sync.Mutex, step *progress.Progress) {
	for item := range avais {
		err := share.SyncRepo(auth, item.local, item.uri, force)
		if err != nil {
			item.status = ERROR
			item.error = err.Error()
		} else {
			item.status = SUCCESS
		}
		*syncRes = append(*syncRes, item)
		step.Advance()
		wg.Done()
	}
}

func availableRepo(repoRes []RepoResult, force string) []RepoResult {
	var avaiRepo []RepoResult
	for _, item := range repoRes {
		if item.status == SUCCESS || (item.status == EXIST && force != SKIP) {
			avaiRepo = append(avaiRepo, item)
		}
	}
	return avaiRepo
}