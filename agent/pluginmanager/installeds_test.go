package pluginmanager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/stretchr/testify/assert"
)

func Test_LoadInstalledPlugins(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := _CorruptedTimeformat
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	defer gomonkey.ApplyFunc(getPreInstalledPluginsBoltPath, func() (string, error) {
		return boltPath, nil
	}).Reset()
	defer gomonkey.ApplyFunc(bolt.Open, func(string, os.FileMode, *bolt.Options) (*bolt.DB, error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)


	plugins, err := loadInstalledPlugins(true)
	assert.Nil(t, plugins)
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}

func TestInstalledPlugins_FindAll(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := "2006-01-02-15:04:05"
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	var db *bolt.DB
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "View", func(*bolt.DB, func(*bolt.Tx) error) (error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)

	plugins := &InstalledPlugins{
		boltdb: &bolt.DB{},
		dbFilePath: boltPath,
	}
	_, _, err := plugins.FindAll()
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}

func TestInstalledPlugins_FindManyByName(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := "2006-01-02-15:04:05"
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	var db *bolt.DB
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "View", func(*bolt.DB, func(*bolt.Tx) error) (error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)

	plugins := &InstalledPlugins{
		boltdb: &bolt.DB{},
		dbFilePath: boltPath,
	}
	_, _, err := plugins.FindManyByName("test")
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}


func TestInstalledPlugins_FindOneWithPredicate(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := "2006-01-02-15:04:05"
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	var db *bolt.DB
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "View", func(*bolt.DB, func(*bolt.Tx) error) (error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)

	plugins := &InstalledPlugins{
		boltdb: &bolt.DB{},
		dbFilePath: boltPath,
	}
	_, _, err := plugins.FindOneWithPredicate(func(plugin *PluginInfo) bool{
		return true
	})
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}


func TestInstalledPlugins_Insert(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := "2006-01-02-15:04:05"
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	defer gomonkey.ApplyFunc(json.Marshal, func(v any) ([]byte, error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)

	plugins := &InstalledPlugins{
		boltdb: &bolt.DB{},
		dbFilePath: boltPath,
	}
	_, err := plugins.Insert(&PluginInfo{})
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}


func TestInstalledPlugins_Update(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := "2006-01-02-15:04:05"
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	defer gomonkey.ApplyFunc(json.Marshal, func(v any) ([]byte, error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)

	plugins := &InstalledPlugins{
		boltdb: &bolt.DB{},
		dbFilePath: boltPath,
	}
	err := plugins.Update(1, &PluginInfo{})
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}



func TestInstalledPlugins_DeleteByKey(t *testing.T) {
	boltDir := os.TempDir()
	boltFile := "mock.db"
	boltPath := filepath.Join(boltDir, boltFile)
	mockTimeStamp := "2006-01-02-15:04:05"
	corruptedPath := filepath.Join(boltDir, "corrupted-and-removeable." + mockTimeStamp + "." + boltFile)

	os.Remove(boltPath)
	os.Remove(corruptedPath)
	defer os.Remove(boltPath)
	defer os.Remove(corruptedPath)

	var tm time.Time
	defer gomonkey.ApplyMethod(reflect.TypeOf(tm), "Format", func(time.Time, string) string {
		return mockTimeStamp
	}).Reset()
	var db *bolt.DB
	defer gomonkey.ApplyMethod(reflect.TypeOf(db), "Update", func(*bolt.DB, func(*bolt.Tx) error) (error) {
		panic("mock panic")
	}).Reset()

	f, e := os.Create(boltPath)
	assert.Nil(t, e)
	f.Close()
	exists := fileutil.CheckFileIsExist(boltPath)
	assert.True(t, exists)

	plugins := &InstalledPlugins{
		boltdb: &bolt.DB{},
		dbFilePath: boltPath,
	}
	err := plugins.DeleteByKey(1)
	assert.Equal(t, "mock panic", err.Error())
	
	exists = fileutil.CheckFileIsExist(corruptedPath)
	assert.True(t, exists)
	exists = fileutil.CheckFileIsExist(boltPath)
	assert.False(t, exists)
}