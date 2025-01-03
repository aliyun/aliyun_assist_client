
package fromexemodel


type ErrorCode string

const (
	CodeGetCACertificateError  ErrorCode = "GetCACertificateError"
	CodeGetServerDomainError   ErrorCode = "GetServerDomainError"
	CodeRegionIdError          ErrorCode = "GetRegionIdError"
	CodeInstanceIdError        ErrorCode = "GetInstanceIdError"
	CodeJSONSerializationError ErrorCode = "JSONSerializationError"
)

type ErrorOutputV1 struct {
	SchemaVersion string   `json:"schemaVersion"`
	Error         []ErrMsg `json:"error"`
}

type ErrMsg struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type ProvisionOutputV1 struct {
	SchemaVersion string `json:"schemaVersion"`
	Result        struct {
		CACertificate    string            `json:"caCertificate"`
		ServerDomain     string            `json:"serverDomain"`
		ExtraHTTPHeaders map[string]string `json:"extraHTTPHeaders"`
		RegionId         string            `json:"regionId"`
	} `json:"result"`
}