package platform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"up2GitX/share"

	"github.com/gookit/gcli/v2"
	"github.com/gookit/color"
	"github.com/gookit/gcli/v2/interact"
)

// options for the command
var nesOpts = struct {
	forceSync bool
}{}

func GiteeCommand() *gcli.Command {
	gitee := &gcli.Command{
		Func:     syncGitee,
		Name:     "gitee",
		UseFor:   "This command is used for sync local repo to Gitee",
		Examples: `Simple usage: <cyan>{$binName} {$cmd} /Users/Zoker/repos/</>`,
	}

	// bind options
	gitee.BoolOpt(&nesOpts.forceSync, "force", "f", false, "Sync local repo to Gitee whether repo exists or not, like git push --force all")

	// bind args with names
	gitee.AddArg("repoDir", "Tell me which repos your want to sync, is required", false)

	return gitee
}

func syncGitee(c *gcli.Command, args []string) error {
	if len(args) == 0 {
		share.InvalidAlert(c.Name)
		return nil
	}

	// todo nesOpts.forceSync
	repos := share.ReadyToAuth(args[0])
	if repos == nil {
		return nil
	}

	askResult, success := askForAccount()
	if !success {
		color.Red.Println(askResult)
		return nil
	}
	accessToken := askResult

	userInfo, success := getUserInfo(accessToken)
	if !success {
		color.Red.Println(userInfo["error"])
		return nil
	}
	color.Green.Printf("Hello, %s! \n", userInfo["name"])
	allNamespace := getNamespace(userInfo)

	namespace := make([]string, len(allNamespace))
	for i, n := range allNamespace {
		namespace[i] = n[1]
	}
	selectedNumber := askNamespace(namespace)
	numberD, _ := strconv.Atoi(selectedNumber)
	selectedNp := allNamespace[numberD]

	fmt.Printf("Selected %s(https://gitee.com/%s) as namespace, Type: %s \n" +
		"The following projects will be generated on Gitee: \n", selectedNp[0], selectedNp[1], selectedNp[2])
	// projects list
	npEnsure, _ := interact.ReadLine("Next step: create projects and sync code, continue?(y/n)")
	if npEnsure != "y" {
		share.ExitMessage()
	}

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