package extid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewExternalMachineId(t *testing.T) {
    payload := "test_payload"
    emi := NewExternalMachineId(payload)
    assert.NotNil(t, emi)
    assert.Equal(t, payload, emi.machineId)
}

func TestExternalMachineIdName(t *testing.T) {
    emi := NewExternalMachineId("")
    assert.NotNil(t, emi)
    assert.Equal(t, "extid", emi.Name())
}

func TestExternalMachineIdGenerate(t *testing.T) {
    cases := []struct {
        payload   string
        expected  string
    }{
        {"machine123", "machine123"},
        {"", ""},
        {"ABC123", "ABC123"},
    }

    for _, c := range cases {
        t.Run(c.payload, func(t *testing.T) {
            emi := NewExternalMachineId(c.payload)
            actual, err := emi.Generate()
            assert.NoError(t, err)
            assert.Equal(t, c.expected, actual)
        })
    }
}
