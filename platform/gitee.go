package platform

import (
	"fmt"
	"up2GitX/share"

	"github.com/gookit/color"
	"github.com/gookit/gcli/v2"
	"github.com/gookit/gcli/v2/interact"
)

// options for the command
var nesOpts = struct {
	forceSync bool
}{}

func GiteeCommand() *gcli.Command {
	gitee := &gcli.Command{
		Func:    syncGitee,
		Name:    "gitee",
		UseFor:  "This command is used for sync local repo to Gitee",
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
		fmt.Printf("Tell me which repos your want to sync, Usage: ")
		color.Cyan.Println("up2 gitee /Users/Zoker/repos/")
		fmt.Println("See 'up2 gitee -h' for more details")
	} else {
		repoDir := args[0]
		// todo nesOpts.forceSync
		if share.DirExists(repoDir) {
			repos, _ := share.GetGitDir(repoDir)
			fmt.Println(repos)
			toGitee, _ := interact.ReadLine("Continue to auth Gitee? (y/n Default: y)")
			fmt.Println(toGitee)
		} else {
			fmt.Println("The path you provided is not a dir or not exists")
		}

		return nil
	}
	return nil
}