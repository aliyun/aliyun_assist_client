package role

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testRoleOutput = "[{\"Name\": \"AD-Certificate\", \"DisplayName\": \"Active Directory Certificate Services\", \"Description\": \"<starte0ca162a>Active Directory Certificate Services (AD CS) is used<endf0d99a4f>\", \"Installed\": \"False\", \"InstalledState\": \"\", \"FeatureType\": \"Role\", \"Path\": \"<starte0ca162a>Active Directory Certificate Services<endf0d99a4f>\", \"SubFeatures\": \"ADCS-Cert-Authority ADCS-Enroll-Web-Pol\", \"ServerComponentDescriptor\": \"ServerComponent_AD_Certificate\", \"DependsOn\": \"\", \"Parent\": \"\"}]"

var testRoleOutputData = []model.RoleData{
	{
		Name:                      "AD-Certificate",
		DisplayName:               "Active Directory Certificate Services",
		Description:               "Active Directory Certificate Services (AD CS) is used",
		Installed:                 "False",
		InstalledState:            "",
		FeatureType:               "Role",
		Path:                      "Active Directory Certificate Services",
		SubFeatures:               "ADCS-Cert-Authority ADCS-Enroll-Web-Pol",
		ServerComponentDescriptor: "ServerComponent_AD_Certificate",
		DependsOn:                 "",
		Parent:                    "",
	},
}

var testServerManagerOutput = `
  		<ServerManagerConfigurationQuery>
			<Role DisplayName="Active Directory Certificate Services" Installed="false" Id="AD-Certificate">
  				<RoleService DisplayName="Certification Authority" Installed="false" Id="ADCS-Cert-Authority" Default="true" >
					<RoleService DisplayName="Certification Authority" Installed="false" Id="ADCS-Cert-Authority" Default="true" />
				</RoleService>
  				<RoleService DisplayName="Certification Authority Web Enrollment" Installed="false" Id="ADCS-Web-Enrollment" />
  				<RoleService DisplayName="Online Responder" Installed="false" Id="ADCS-Online-Cert" />
  				<RoleService DisplayName="Network Device Enrollment Service" Installed="true" Id="ADCS-Device-Enrollment" />
			</Role>

			<Feature DisplayName="BitLocker Drive Encryption" Installed="false" Id="BitLocker" />
			<Role DisplayName="NotActive Directory Certificate Services" Installed="false" Id="AD-Certificate">
  				<RoleService DisplayName="Certification Authority" Installed="false" Id="ADCS-Cert-Authority" Default="true" />
  				<RoleService DisplayName="Certification Authority Web Enrollment" Installed="false" Id="ADCS-Web-Enrollment" />
  				<RoleService DisplayName="Online Responder" Installed="false" Id="ADCS-Online-Cert" />
  				<RoleService DisplayName="Network Device Enrollment Service" Installed="false" Id="ADCS-Device-Enrollment" />
			</Role>
			<Feature DisplayName="Remote Server Administration Tools" Installed="false" Id="RSAT">
      			<Feature DisplayName="Role Administration Tools" Installed="false" Id="RSAT-Role-Tools">
         			<Feature DisplayName="Active Directory Certificate Services Tools" Installed="false" Id="RSAT-ADCS"/>
            		<Feature DisplayName="Certification Authority Tools" Installed="false" Id="RSAT-ADCS-Mgmt" />
            		<Feature DisplayName="Online Responder Tools" Installed="false" Id="RSAT-Online-Responder" />
         		</Feature>
			</Feature>
  		</ServerManagerConfigurationQuery>
  	`

func createMockTestExecuteCommand(output string, err error) func(string, ...string) ([]byte, error) {

	return func(string, ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func createMockReadAllText(output string, err error) func(string) (string, error) {
	return func(string) (string, error) {
		return output, err
	}
}

func TestGetRoleData(t *testing.T) {

	cmdExecutor = createMockTestExecuteCommand(testRoleOutput, nil)
	startMarker = "<starte0ca162a>"
	endMarker = "<endf0d99a4f>"
	data, err := collectRoleData(model.Config{})

	assert.Nil(t, err)
	assert.Equal(t, testRoleOutputData, data)
}

func TestGetRoleDataUsingServerManager(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("testOutput", nil)
	readFile = createMockReadAllText(testServerManagerOutput, nil)

	var roleInfo []model.RoleData

	err := collectDataUsingServerManager(&roleInfo)

	assert.Nil(t, err)
	assert.Equal(t, 17, len(roleInfo))
	assert.Equal(t, "AD-Certificate", roleInfo[0].Name)
	assert.Equal(t, "False", roleInfo[0].Installed)
	assert.Equal(t, "Active Directory Certificate Services", roleInfo[0].DisplayName)
}

func TestGetRoleDataUsingServerManagerCmdError(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("testOutput", errors.New("Error"))
	readFile = createMockReadAllText(testServerManagerOutput, nil)

	var roleInfo []model.RoleData

	err := collectDataUsingServerManager(&roleInfo)

	assert.NotNil(t, err)
}

func TestGetRoleDataUsingServerManagerXmlErr(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("testOutput", nil)
	readFile = createMockReadAllText("unexpected output", nil)

	var roleInfo []model.RoleData

	err := collectDataUsingServerManager(&roleInfo)

	assert.NotNil(t, err)
}

func TestGetRoleDataUsingServerManagerReadError(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("testOutput", nil)
	readFile = createMockReadAllText("", errors.New("error"))

	var roleInfo []model.RoleData

	err := collectDataUsingServerManager(&roleInfo)

	assert.NotNil(t, err)
}

func TestGetRoleDataUsingServerManagerFilePathError(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("testOutput", nil)
	readFile = createMockReadAllText("", errors.New("error"))

	var roleInfo []model.RoleData

	err := collectDataUsingServerManager(&roleInfo)

	assert.NotNil(t, err)
}

func TestGetRoleDataCmdExeError(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("", errors.New("error"))
	startMarker = "<starte0ca162a>"
	endMarker = "<endf0d99a4f>"

	data, err := collectRoleData(model.Config{})

	assert.NotNil(t, err)
	assert.Nil(t, data)
}

func TestGetRoleDataCmdUnexpectedOutput(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("invalid output", nil)
	startMarker = "<starte0ca162a>"
	endMarker = "<endf0d99a4f>"

	data, err := collectRoleData(model.Config{})

	assert.NotNil(t, err)
	assert.Nil(t, data)
}

func TestGetRoleDataInvalidMarkedFields(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand(testRoleOutput, nil)
	startMarker = "<starte0ca162a>"

	endMarker = "<test>"

	data, err := collectRoleData(model.Config{})

	assert.NotNil(t, err)
	assert.Nil(t, data)
}
