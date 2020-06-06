package platform

import (
	"encoding/json"
	"fmt"
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
	} else {
		// todo nesOpts.forceSync
		repos := share.ReadyToAuth(args[0])
		if repos != nil {
			askResult, success := askForAccount()
			if success {
				fmt.Println(askResult)
			} else {
				color.Red.Println(askResult)
			}
		}
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
					"client_id": "xxxxxxx",
					"client_secret": "xxxxxxxx",
					"scope": "projects groups enterprises"
					}`, email, password)

		var paramsJson map[string]interface{}
		json.Unmarshal([]byte(params), &paramsJson)
		result, err := share.Post("https://gitee.com/oauth/token", paramsJson)

		if err != nil {
			return err.Error(), false
		}

		accessToken, atok := result["access_token"].(string)
		_, errok := result["error"].(string)
		if atok {
			return accessToken, true
		} else if errok {
			return result["error_description"].(string), false
		}
	}
	return "Unexpectedly exit", false
}
