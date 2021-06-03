package model

import "strings"

const (
	// ACSInstanceInformation is inventory type of instance information
	ACSInstanceInformation = "ACS:InstanceInformation"
	// Enabled represents constant string used to enable various components of inventory plugin
	Enabled = "Enabled"
	// ErrorThreshold represents error threshold for inventory plugin
	ErrorThreshold = 10
	// InventoryPolicyDocName represents name of inventory policy doc
	InventoryPolicyDocName = "policy.json"
	// SizeLimitKBPerInventoryType represents size limit in KB for 1 inventory data type
	// Bump up to 3MB for agent. We have more strict size limit rule in the micro service.
	SizeLimitKBPerInventoryType = 3072
	// TotalSizeLimitKB represents size limit in KB for 1 PutInventory API call
	TotalSizeLimitKB = 10240
	// Standard name for 64-bit architecture
	Arch64Bit = "x86_64"
	// Standard name for 32-bit architecture
	Arch32Bit = "i386"
)

// Item encapsulates an inventory item
type Item struct {
	Name string
	//content depends on inventory type - hence set as interface{} here.
	//e.g: for application - it will contain []ApplicationData,
	//for instanceInformation - it will contain []InstanceInformation.
	Content       interface{}
	ContentHash   string
	SchemaVersion string
	CaptureTime   string
}

// InstanceInformation captures all attributes present in ACS:InstanceInformation inventory type
type InstanceInformation struct {
	AgentName       string
	AgentVersion    string
	ComputerName    string
	PlatformName    string
	PlatformType    string
	PlatformVersion string
	InstanceId      string
	IpAddress       string
	ResourceType    string
	RamRole         string
}

// ApplicationData captures all attributes present in ACS:Application inventory type
type ApplicationData struct {
	ApplicationType string `json:",omitempty"`
	Architecture    string
	Epoch           string `json:",omitempty"`
	InstalledTime   string `json:",omitempty"`
	Name            string
	PackageId       string `json:",omitempty"`
	Publisher       string
	Release         string `json:",omitempty"`
	Summary         string `json:",omitempty"`
	URL             string `json:",omitempty"`
	Version         string
}

// FileData captures all attributes present in ACS:File inventory type
type FileData struct {
	CompanyName      string
	Description      string
	FileVersion      string
	InstalledDate    string
	InstalledDir     string
	LastAccessTime   string
	ModificationTime string
	Name             string
	ProductLanguage  string
	ProductName      string
	ProductVersion   string
	Size             string
}

// NetworkData captures all attributes present in ACS:Network inventory type
type NetworkData struct {
	DHCPServer string `json:",omitempty"`
	DNSServer  string `json:",omitempty"`
	Gateway    string `json:",omitempty"`
	IPV4       string
	IPV6       string
	MacAddress string
	Name       string
	SubnetMask string `json:",omitempty"`
}

type RoleData struct {
	DependsOn                 string
	Description               string
	DisplayName               string
	FeatureType               string
	Installed                 string
	InstalledState            string
	Name                      string
	Parent                    string
	Path                      string
	ServerComponentDescriptor string
	SubFeatures               string
}

type ServiceData struct {
	DependentServices  string
	DisplayName        string
	Name               string
	ServiceType        string
	ServicesDependedOn string
	StartType          string
	Status             string
}

type RegistryData struct {
	KeyPath   string
	Value     string
	ValueName string
	ValueType string
}

type WindowsUpdateData struct {
	Description   string
	HotFixId      string
	InstalledBy   string
	InstalledTime string
}

// InstanceDetailedInformation captures all attributes present in ACS:InstanceDetailedInformation inventory type
type InstanceDetailedInformation struct {
	CPUCores              string
	CPUHyperThreadEnabled string
	CPUModel              string
	CPUSockets            string
	CPUSpeedMHz           string
	CPUs                  string
	OSServicePack         string
}

// FormatArchitecture converts different architecture values to the standard inventory value
func FormatArchitecture(arch string) string {
	arch = strings.ToLower(strings.TrimSpace(arch))
	if arch == "amd64" {
		return Arch64Bit
	}
	if arch == "386" {
		return Arch32Bit
	}
	return arch
}

type Config struct {
	Collection string `json:"Collection"`
	Filters    string `json:"Filters"`
	Location   string `json:"Location"`
}

type Policy struct {
	InventoryPolicy map[string]Config `json:"Policy"`
}

type InventoryItem struct {
	// CaptureTime is a required field
	CaptureTime *string `type:"string" required:"true"`

	// The inventory data of the inventory type.
	Content []map[string]*string `type:"list"`

	// MD5 hash of the inventory item type contents. The content hash is used to
	// determine whether to update inventory information. The PutInventory API does
	// not update the inventory item type contents if the MD5 hash has not changed
	// since last update.
	ContentHash *string `type:"string"`

	// The schema version for the inventory item.
	//
	// SchemaVersion is a required field
	SchemaVersion *string `type:"string" required:"true"`
	//
	// TypeName is a required field
	TypeName *string `min:"1" type:"string" required:"true"`
}

type PutInventoryInput struct {

	// An instance ID where you want to add or update inventory items.
	//
	// InstanceId is a required field
	InstanceId *string `type:"string" required:"true"`

	// The inventory items that you want to add or update on instances.
	//
	// Items is a required field
	Items []*InventoryItem `min:"1" type:"list" required:"true"`
}
