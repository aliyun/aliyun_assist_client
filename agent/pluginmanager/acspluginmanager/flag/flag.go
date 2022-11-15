package flag

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
)

const (
	HelpFlagName          = "help"
	VerboseFlagName       = "verbose"
	VersionFlagName       = "version"
	ListFlagName          = "list"
	LocalFlagName         = "local"
	VerifyFlagName        = "verify"
	StatusFlagName        = "status"
	PluginFlagName        = "plugin"
	PluginIdFlagName      = "pluginId"
	PluginVersionFlagName = "pluginVersion"
	ParamsFlagName        = "params"
	ParamsV2FlagName      = "paramsV2"
	UrlFlagName           = "url"
	SeparatorFlagName     = "separator"
	FileFlagName          = "file"
	ExecFlagName          = "exec"
	RemoveFlagName        = "remove"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(NewHelpFlag())
	fs.Add(NewVersionFlag())
	fs.Add(NewLocalFlag())
	fs.Add(NewPluginFlag())
	fs.Add(NewPluginIdFlag())
	fs.Add(NewPluginVersionFlag())
	fs.Add(NewParamsFlag())
	fs.Add(NewParamsV2Flag())
	fs.Add(NewUrlFlag())
	fs.Add(NewSeparatorFlag())
	fs.Add(NewFileFlag())
	fs.Add(NewVerboseFlag())
	fs.Add(NewListFlag())
	fs.Add(NewVerifyFlag())
	fs.Add(NewStatusFlag())
	fs.Add(NewExecFlag())
	fs.Add(NewRemoveFlag())
}

func VerboseFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(VerboseFlagName)
}

func ListFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ListFlagName)
}

func LocalFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(LocalFlagName)
}

func VersionFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(VersionFlagName)
}

func VerifyFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(VerifyFlagName)
}

func StatusFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(StatusFlagName)
}

func PluginFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(PluginFlagName)
}

func PluginIdFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(PluginIdFlagName)
}

func PluginVersionFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(PluginVersionFlagName)
}

func ParamsFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ParamsFlagName)
}

func ParamsV2Flag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ParamsV2FlagName)
}

func UrlFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(UrlFlagName)
}

func SeparatorFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(SeparatorFlagName)
}

func FileFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(FileFlagName)
}

func ExecFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ExecFlagName)
}

func RemoveFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RemoveFlagName)
}

func NewHelpFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         HelpFlagName,
		Shorthand:    'h',
		AssignedMode: cli.AssignedNone,
		Persistent:   true,
		Short: i18n.T(
			`--help, print this page`,
			`--help, 打印此帮助页`,
		),
	}
}

func NewVerboseFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         VerboseFlagName,
		Shorthand:    'V',
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Persistent:   true,
		Short: i18n.T(
			`--verbose, log more`,
			`--verbose, 打印更多的日志`,
		),
	}
}

func NewVersionFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         VersionFlagName,
		Shorthand:    'v',
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Persistent:   true,
		Short: i18n.T(
			`--version, print version`,
			`--version, 打印版本号`,
		),
	}
}

func NewListFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ListFlagName,
		Shorthand:    'l',
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Short: i18n.T(
			`--list, show all plugins
	--list --local, only show installed plugins`,
			`--list, 列出所有插件
	--list --local, 只列出本地已安装的插件`,
		),
	}
}

func NewLocalFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         LocalFlagName,
		Shorthand:    'L',
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Short: i18n.T(
			` `,
			` `,
		),
	}
}

func NewVerifyFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         VerifyFlagName,
		Shorthand:    'f',
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Short: i18n.T(
			`--verify --url <> --params <>, verify plugin`,
			`--verify --url <> --params <>, 验证插件`)}
}

func NewStatusFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         StatusFlagName,
		Shorthand:    'S',
		AssignedMode: cli.AssignedNone,
		DefaultValue: "",
		Short: i18n.T(
			`--status, print all plugins status`,
			`--status, 打印所有插件的状态`)}
}

func NewPluginFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         PluginFlagName,
		Shorthand:    'P',
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
		Short: i18n.T(
			`select plugin by name`,
			`通过名称指定插件`)}
}

func NewPluginIdFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         PluginIdFlagName,
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
		Short: i18n.T(
			"select plugin by id",
			"通过插件id指定插件")}
}

///////////////////////////////////////////////////////////////////////////////////////////
//--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair|RamRoleArnWithRoleName}
func NewPluginVersionFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         PluginVersionFlagName,
		Shorthand:    'n',
		AssignedMode: cli.AssignedOnce,
		DefaultValue: "",
		Short: i18n.T(
			"set plugin version",
			"指定插件版本号")}
}

func NewParamsFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ParamsFlagName,
		Shorthand:    'p',
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"set params for plugin with separator, set separator by --separator",
			"设置插件的运行参数，通过--separator设置参数的分隔符")}
}

func NewParamsV2Flag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ParamsV2FlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"set params for plugin without separator",
			"设置插件的运行参数，不使用分隔符分割")}
}

func NewUrlFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         UrlFlagName,
		Shorthand:    'u',
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"set plugin's url",
			"指定插件地址")}
}

func NewSeparatorFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         SeparatorFlagName,
		Shorthand:    's',
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"set separator to split plugin's params, default is [,]",
			"指定插件参数的分隔符，默认为逗号")}
}

func NewFileFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         FileFlagName,
		Shorthand:    'F',
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"select plugin file",
			"指定插件文件")}
}

func NewExecFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ExecFlagName,
		Shorthand:    'e',
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			`--exec --plugin <> --params <>, execute plugin online
	--exec --local --plugin <> --params <>, execute plugin from local
	--exec --file <> --params <>, execute plugin from file`,
			`--exec --plugin <> --params <>, 执行插件，优先从线上查找插件
	--exec --local --plugin <> --params <>, 仅从本地查找执行插件
	--exec --file <> --params <>, 从插件包文件执行插件`)}
}

func NewRemoveFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         RemoveFlagName,
		Shorthand:    'r',
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			`--remove --plugin <>, remove local plugin, will delete plugin's directories`,
			`--remove --plugin <>, 移除本地插件，会删除掉该插件的目录文件`)}
}
