package hybrid

type RegisterInfo struct {
	Code            string `json:"activationCode"`
	MachineId       string `json:"machineId"`
	Fingerprint     string `json:"fingerprint"`
	RegionId        string `json:"regionId"`
	InstanceName    string `json:"instanceName"`
	Hostname        string `json:"hostname"`
	IntranetIp      string `json:"intranetIp"`
	OsVersion       string `json:"osVersion"`
	OsType          string `json:"osType"`
	ClientVersion   string `json:"agentVersion"`
	PublicKeyBase64 string `json:"publicKey"`
	Id              string `json:"activationId"`
	Tag             []Tag  `json:"tag"`
}

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type registerResponse struct {
	Code       int    `json:"code"`
	InstanceId string `json:"instanceId"`
	ErrCode    string `json:"errCode"`
	ErrMsg     string `json:"errMsg"`
}

type unregisterResponse struct {
	Code int `json:"code"`
}

type updateFingerprintRequest struct {
	OldFingerprint string `json:"oldFingerprint"`
	NewFingerprint string `json:"newFingerprint"`
	PublicKey      string `json:"publicKey"`
	InstanceId     string `json:"instanceId"`
	HostName       string `json:"hostname"`
	OsType         string `json:"osType"`
}

type updateFingerprintResponse struct {
	Code    int    `json:"code"`
	ErrCode string `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
}
