package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	logrusr "github.com/aliyun/aliyun_assist_client/thirdparty/bombsimon/logrusr/v3"
	"github.com/rodaine/table"
	"k8s.io/klog/v2"

	"github.com/aliyun/aliyun_assist_client/agent/container"
	"github.com/aliyun/aliyun_assist_client/agent/container/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
)

type ErrorRespond struct {
	ErrorMessage string `json:"error"`
}

const (
	AllFlagName     = "all"
	JsonFlagName    = "json"
	SourceFlagName  = "source"
	TimeoutFlagName = "timeout"
)

var (
	defaultDataSourceName = "all"
	defaultTimeout        = 2 * time.Second

	listContainersFlags = []cli.Flag{
		{
			Name:         AllFlagName,
			Shorthand:    'a',
			Short:        i18n.T(`show all containers. Only running containers are shown by default`, `列出所有容器。默认只列出正在运行的容器`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         JsonFlagName,
			Short:        i18n.T(`print container list in JSON format`, `以JSON格式打印容器列表`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:         SourceFlagName,
			Shorthand:    's',
			Short:        i18n.T(`set source interface from which containers are listed. Possibles are "cri", "docker", or "all". Default: "all"`, `指定列出容器的数据来源，可选的值有 "cri"、"docker" 或者全部列出 "all"。默认为 "all"`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:      TimeoutFlagName,
			Shorthand: 't',
			Short: i18n.T(`timeout of connecting to the container runtime service in seconds, e.g., 2s or 10s. 0 or less would be set to 2s by default`,
				`指定连接容器运行时服务时的超时时间，以秒为单位，如 2s 或者 10s。指定 0 或者负数时采用默认值 2s`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
	}

	listContainersCmd = cli.Command{
		Name:              "list-containers",
		Short:             i18n.T("List containers on the instance", "列出该实例上的容器"),
		Usage:             "list-containers [flags]",
		Sample:            "",
		EnableUnknownFlag: false,
		Run:               runListContainersCmd,
	}
)

func init() {
	for j := range listContainersFlags {
		listContainersCmd.Flags().Add(&listContainersFlags[j])
	}
}

func runListContainersCmd(ctx *cli.Context, args []string) error {
	// Extract value of persistent flags
	logPath, _ := ctx.Flags().Get(LogPathFlagName).GetValue()
	// Extract value of flags just for the command
	showAllContainers := ctx.Flags().Get(AllFlagName).IsAssigned()
	useJsonFormat := ctx.Flags().Get(JsonFlagName).IsAssigned()

	connectTimeout := defaultTimeout
	if value, assigned := ctx.Flags().Get(TimeoutFlagName).GetValue(); assigned {
		var err error
		connectTimeout, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	}
	dataSourceName := defaultDataSourceName
	if value, assigned := ctx.Flags().Get(SourceFlagName).GetValue(); assigned {
		switch value {
		case "all", "cri", "docker":
			dataSourceName = value
		case "":
			return fmt.Errorf(`Specified source must not be empty and should be one of "all", "cri" or "docker"`)
		default:
			return fmt.Errorf(`Specified source is neither of "all", "cri" or "docker": %s`, value)
		}
	}

	// Necessary initialization work
	log.InitLog("aliyun_assist_main.log", logPath)
	// Redirect logging messages from kubernetes CRI client via klog to logrus
	// used by ourselves
	klog.SetLogger(logrusr.New(log.GetLogger()).WithName("klog"))
	e := PatchGolang()
	if e != nil {
		log.GetLogger().Fatal("PatchGolang failed :", e.Error())
	}

	containers, err := container.ListContainers(container.ListContainersOptions{
		ConnectTimeout:    connectTimeout,
		DataSourceName:    dataSourceName,
		ShowAllContainers: showAllContainers,
	})
	if len(containers) == 0 && err != nil {
		return printErrorOrReturn(err, useJsonFormat)
	}

	if useJsonFormat {
		jsonBytes, err := json.Marshal(containers)
		if err != nil {
			return printErrorOrReturn(err, useJsonFormat)
		}

		fmt.Println(string(jsonBytes))
	} else {
		printContainerListText(containers)
	}

	return nil
}

func printErrorOrReturn(err error, useJsonFormat bool) error {
	if !useJsonFormat {
		return err
	}

	jsonBytes, marshalErr := json.Marshal(ErrorRespond{
		ErrorMessage: err.Error(),
	})
	if marshalErr != nil {
		return err
	}

	fmt.Fprintln(os.Stderr, string(jsonBytes))
	return nil
}

func printContainerListText(containers []model.Container) {
	tbl := table.New("Container Id", "Container Name", "Pod Name", "Runtime", "State", "Data Source")
	for _, c := range containers {
		tbl.AddRow(c.Id, c.Name, c.PodName, c.RuntimeName, c.State, c.DataSource)
	}
	tbl.Print()
}
