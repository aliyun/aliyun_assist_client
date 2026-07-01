package log

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	rotatelogs "github.com/aliyun/aliyun_assist_client/thirdparty/file-rotatelogs"
	log "github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/pkg/errors"
)

var Log *log.Logger
var defaultLevel log.Level = log.InfoLevel
var (
	_rotationCount uint  = 365                //max value
	_rotationSize  int64 = 1024 * 1024 * 1024 //max value
	_rotateLog     *rotatelogs.RotateLogs
	rotateLock     sync.Mutex

	_filename            string
	_logpath             string
	_ignoreRotationError bool
)

type Fields = log.Fields

// DefaultCommonFields returns preset fields for each log message. All default
// common fields MUST be prefixed by an underscore to avoid conflict with fields
// added later.
func DefaultCommonFields() log.Fields {
	return log.Fields{
		"_pid": os.Getpid(),
	}
}

func InitLog(filename string, logpath string, ignoreRotationError bool) {
	if logpath != "" {
		pathutil.SetLogPath(filepath.Join(logpath, "log"))
	}
	logdir, _ := pathutil.GetLogPath()

	writer, err := rotatelogs.New(
		filepath.Join(logdir, filename+".%Y%m%d"),
		rotatelogs.WithMaxAge(time.Duration(24*30)*time.Hour),    //最长保留30天
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour), //每天进行一次日志切割
		rotatelogs.WithLinkName(filepath.Join(logdir, filename)), // 为日志文件创建一个名字不变的链接
		rotatelogs.IgnoreRotationError(ignoreRotationError),
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}
	_rotateLog = writer

	Log = log.New()
	Log.SetFormatter(&CustomLogrusTextFormatter{
		CommonFields: DefaultCommonFields(),
	})
	Log.SetOutput(writer)
	Log.SetLevel(defaultLevel)
}

func InitLogWithRotationParams(filename string, logpath string, ignoreRotationError bool) {
	rotateLock.Lock()
	defer rotateLock.Unlock()

	_filename = filename
	_logpath = logpath
	_ignoreRotationError = ignoreRotationError
	NewRotateLog()

	Log = log.New()
	Log.SetFormatter(&CustomLogrusTextFormatter{
		CommonFields: DefaultCommonFields(),
	})
	Log.SetOutput(_rotateLog)
	Log.SetLevel(defaultLevel)
}

func NewRotateLog() {
	if _logpath != "" {
		pathutil.SetLogPath(filepath.Join(_logpath, "log"))
	}
	logdir, _ := pathutil.GetLogPath()
	writer, err := rotatelogs.New(
		filepath.Join(logdir, _filename+".%Y%m%d"),
		rotatelogs.WithRotationCount(_rotationCount),
		rotatelogs.WithRotationSize(_rotationSize),
		rotatelogs.WithRotationTime(time.Duration(24)*time.Hour),  //每天进行一次日志切割
		rotatelogs.WithLinkName(filepath.Join(logdir, _filename)), // 为日志文件创建一个名字不变的链接
		rotatelogs.IgnoreRotationError(_ignoreRotationError),
	)
	if err != nil {
		log.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	} else { //更新 _rotateLog、Log、最后关闭旧的
		oldWriter := _rotateLog
		_rotateLog = writer

		if Log != nil {
			Log.SetOutput(writer)
		}

		if oldWriter != nil {
			err = oldWriter.Close()
			if err != nil {
				log.Errorf("close oldWriter for log failed. %+v", errors.WithStack(err))
			}
		}
	}
}

// Update logFileCountLimit
func UpdateLogFileCountLimit(logger log.FieldLogger, v any) {
	rotateLock.Lock()
	defer rotateLock.Unlock()

	if _rotateLog == nil {
		logger.Errorf("_rotateLog is nil, please check.")
	}
	logger.Infof("UpdateLogFileCountLimit with %+v", v)

	val, ok := v.(int64)
	if !ok || val < 0 {
		logger.Errorf("UpdateLogFileCountLimit expect int64 and >0, got %T", v)
		return
	}
	_rotationCount = uint(val)

	NewRotateLog()
}

// Update logSizeLimit
func UpdateLogSizeLimit(logger log.FieldLogger, v any) {
	rotateLock.Lock()
	defer rotateLock.Unlock()

	if _rotateLog == nil {
		logger.Errorf("_rotateLog is nil, please check.")
	}
	logger.Infof("UpdateLogSizeLimit with %+v", v)
	val, ok := v.(int64)
	if !ok || val < 0 {
		logger.Errorf("UpdateLogSizeLimit expect int64 and >0, got %T", v)
		return
	}

	_rotationSize = int64(val)

	NewRotateLog()
}

func GetLogger() *log.Logger {
	if Log == nil {
		InitLog("aliyun_assist_test", "", false)
	}
	return Log
}
