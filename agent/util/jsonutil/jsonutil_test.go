package jsonutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type JsonstrTest struct {
	Test  string `json:"test"`
	Test1 string
}

func TestIndent(t *testing.T) {
	jsonstr := `{"test":"test", "test1":"test1"}`
	val := Indent(jsonstr)
	print(val)
	assert.Equal(t, "{\n  \"test\": \"test\",\n  \"test1\": \"test1\"\n}", val)
}

func TestMarshal(t *testing.T) {
	testResp := JsonstrTest{
		Test:  "test",
		Test1: "test1",
	}
	res, _ := Marshal(testResp)
	print(res)
	assert.Equal(t, `{"test":"test","Test1":"test1"}`+"\n", res)
}

func TestRemarshal(t *testing.T) {
	cred := JsonstrTest{
		Test:  "test",
		Test1: "test1",
	}
	cred1 := &JsonstrTest{}
	Remarshal(cred, &cred1)
	assert.Equal(t, "test", cred1.Test)
	assert.Equal(t, "test1", cred1.Test1)

}

func TestUnmarshal(t *testing.T) {
	jsonstr := "{\"test\":\"test\", \"test1\":\"test1\"}"

	cred := &JsonstrTest{}
	Unmarshal(jsonstr, &cred)
	assert.Equal(t, "test", cred.Test)
	assert.Equal(t, "test1", cred.Test1)
}

func TestMarshalIndent(t *testing.T) {
	cred := JsonstrTest{
		Test:  "test",
		Test1: "test1",
	}
	result, _ := MarshalIndent(cred)
	assert.Equal(t, "{\n  \"test\": \"test\",\n  \"Test1\": \"test1\"\n}", result)
}
