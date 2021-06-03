package role

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testRole = []model.RoleData{
	{
		Name:                      "AD-Certificate",
		DisplayName:               "Active Directory Certificate Services",
		Description:               "Active Directory Certificate Services (AD CS) is used",
		Installed:                 "False",
		InstalledState:            "",
		FeatureType:               "Role",
		Path:                      "Active Directory Certificate Services",
		SubFeatures:               "ADCS-Cert-Authority ADCS-Enroll-Web-Pol ADCS-Enroll-Web-Svc",
		ServerComponentDescriptor: "ServerComponent_AD_Certificate",
		DependsOn:                 "",
		Parent:                    "",
	},
	{
		Name:                      "ADCS-Cert-Authority",
		DisplayName:               "Certification Authority",
		Description:               "Certification Authority (CA) is used to issue and manage certificates.",
		Installed:                 "False",
		InstalledState:            "",
		FeatureType:               "Role Service",
		Path:                      "Active Directory Certificate Services\\Certification Authority",
		SubFeatures:               "",
		ServerComponentDescriptor: "ServerComponent_ADCS_Cert_Authority",
		DependsOn:                 "",
		Parent:                    "AD-Certificate",
	},
}

func testCollectRoleData(config model.Config) (data []model.RoleData, err error) {
	return testRole, nil
}

func TestGatherer(t *testing.T) {
	gatherer := Gatherer()
	collectData = testCollectRoleData
	item, err := gatherer.Run(model.Config{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(item))
	assert.Equal(t, GathererName, item[0].Name)
	assert.Equal(t, SchemaVersionOfRoleGatherer, item[0].SchemaVersion)
	assert.Equal(t, testRole, item[0].Content)
}
