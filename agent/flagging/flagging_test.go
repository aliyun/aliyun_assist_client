package flagging

import (
	"os"
	"path/filepath"
	"go.uber.org/atomic"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util/ramflag"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/stretchr/testify/assert"
)

func TestIsColdstart(t *testing.T) {
	// delete ram flag
	ramflag.Delete(coldstartFlagName)
	isColdstart, err := IsColdstart()
	assert.Nil(t, err)
	assert.True(t, isColdstart)

	// ram flag exists but isColdstart is still true because IsColdstart re-use
	// previous result
	isColdstart, err = IsColdstart()
	assert.Nil(t, err)
	assert.True(t, isColdstart)

	// IsColdstart will check ram falg if _isColdstartFlaggedOnStartup is nil
	// IsColdstart return false because the ram flag exist
	_isColdstartFlaggedOnStartup = nil
	isColdstart, err = IsColdstart()
	assert.Nil(t, err)
	assert.False(t, isColdstart)
}

func Test_detectFlagSet(t *testing.T) {
	currentDir, err := os.Executable()
	assert.Nil(t, err)
	currentDir = filepath.Dir(currentDir)
	confDir := filepath.Join(currentDir, "conf")
	crossVersionConfDir := filepath.Join(currentDir, "crossversionconf")
	os.Mkdir(confDir, os.ModePerm)
	os.Mkdir(crossVersionConfDir, os.ModePerm)
	defer os.RemoveAll(confDir)
	defer os.RemoveAll(crossVersionConfDir)

	guard_1 := gomonkey.ApplyFunc(pathutil.GetConfigPath, func() (string, error) {
		return confDir, nil
	})
	defer guard_1.Reset()
	guard_2 := gomonkey.ApplyFunc(pathutil.GetCrossVersionConfigPath, func() (string, error) {
		return crossVersionConfDir, nil
	})
	defer guard_2.Reset()

	testflags := []struct {
		Name string
		FileName string
		Flag *atomic.Bool
	}{
		{
			Name: disableNormalizeCRLFFlagName,
			FileName: disableNormalizeCRLFFlagFilename,
			Flag: &_normalizingCRLFDisabled,
		},
		{
			Name: disableTaskOutputRingbufferFlagName,
			FileName: disableTaskOutputRingbufferFlagFilename,
			Flag: &_taskOutputRingbufferDisabled,
		},
	}

	for _, tf := range testflags {
		res, err := detectFlagSet(tf.Name, tf.FileName, tf.Flag)
		assert.Nil(t, err)
		assert.False(t, res)
		tf.Flag.Store(false)

		f, err := os.Create(filepath.Join(confDir, tf.FileName))
		assert.Nil(t, err)
		f.Close()
		res, err = detectFlagSet(tf.Name, tf.FileName, tf.Flag)
		assert.Nil(t, err)
		assert.True(t, res)
		assert.True(t, tf.Flag.Load())
		assert.Nil(t, os.Remove(filepath.Join(confDir, tf.FileName)))

		res, err = detectFlagSet(tf.Name, tf.FileName, tf.Flag)
		assert.Nil(t, err)
		assert.False(t, res)
		tf.Flag.Store(false)

		f, err = os.Create(filepath.Join(crossVersionConfDir, tf.FileName))
		assert.Nil(t, err)
		f.Close()
		res, err = detectFlagSet(tf.Name, tf.FileName, tf.Flag)
		assert.Nil(t, err)
		assert.True(t, res)
		assert.True(t, tf.Flag.Load())
		assert.Nil(t, os.Remove(filepath.Join(crossVersionConfDir, tf.FileName)))

		res, err = detectFlagSet(tf.Name, tf.FileName, tf.Flag)
		assert.Nil(t, err)
		assert.False(t, res)
		tf.Flag.Store(false)
	}
	
	
}
