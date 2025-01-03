package model

var (
	commanderName                string
	commanderSupportedApiVersion string
)

func GetCommanderBaseInfo() (CommanderName string, CommanderSupportedApiVersion string) {

	return commanderName, commanderSupportedApiVersion
}

func SetCommanderBaseInfo(name string, apiVersion string) {
	commanderName = name
	commanderSupportedApiVersion = apiVersion
}
