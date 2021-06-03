package resources

const (
	Compliant    = "Compliant"
	NotCompliant = "NotCompliant"
	Failed       = "Failed"
)

type ResourceState interface {
	Load(properties map[string]interface{}) (err error)
	Apply() (status string, extraInfo string, err error)
	Monitor() (status string, extraInfo string, err error)
}
