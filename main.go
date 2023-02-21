//go:generate goversioninfo -o=resource_windows.syso

package main

import (
	"os"

	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
)

var (
	gitHash   string
	assistVer string = "10.10.10.10000"
)

func main() {
	cli.PlatformCompatible()

	i18n.SetLanguage("en")

	ctx := cli.NewCommandContext(cli.DefaultWriter())
	ctx.EnterCommand(&rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())

	rootCmd.AddSubCommand(&listContainersCmd)
	rootCmd.AddSubCommand(&dataEncryptionCmd)

	rootCmd.Execute(ctx, os.Args[1:])
}
