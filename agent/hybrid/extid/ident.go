package extid

type ExternalMachineId struct {
	machineId string
}

func NewExternalMachineId(payload string) *ExternalMachineId {
	return &ExternalMachineId{
		machineId: payload,
	}
}

func (*ExternalMachineId) Name() string {
	return "extid"
}

func (emi *ExternalMachineId) Generate() (string, error) {
	return emi.machineId, nil
}
