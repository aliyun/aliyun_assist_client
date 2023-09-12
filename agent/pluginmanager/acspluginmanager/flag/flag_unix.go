//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package flag

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
)

const (
	ExperimentalForegroundFlagName = "experimental-foreground"
)

func AddPlatformSpecificFlags(fs *cli.FlagSet) {
	fs.Add(NewExperimentalForegroundFlag())
}

func ExperimentalForegroundFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ExperimentalForegroundFlagName)
}

func NewExperimentalForegroundFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ExperimentalForegroundFlagName,
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Persistent:   true,
		Short: i18n.T(
			`(experimental) make plugin interactive when running in terminal`,
			`（实验特性）支持插件运行时在当前所在的终端上进行交互`,
		),
	}
}
