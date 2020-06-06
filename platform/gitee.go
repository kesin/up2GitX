package platform

import (
	"up2GitX/share"

	"github.com/gookit/gcli/v2"
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
		share.ReadyToAuth(args[0])
	}
	return nil
}
