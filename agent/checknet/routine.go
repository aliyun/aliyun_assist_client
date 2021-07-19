package checknet

type NetcheckRequestType string
const (
	NetcheckRequestNormal NetcheckRequestType = "normal"
	NetcheckRequestForceOnce NetcheckRequestType = "forceOnce"
)
