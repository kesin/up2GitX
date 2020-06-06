package main

import (
	"up2GitX/platform"

	"github.com/gookit/gcli/v2"
)

func main() {
	up2 := gcli.NewApp()
	up2.Version = "1.0.0"
	up2.Description = "A tool for easily sync multiple local repo to different platform like Gitee, Github or Gitlab"
	up2.Add(platform.GiteeCommand())
	up2.Run()
}
