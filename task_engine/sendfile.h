// FileOp.h: 标准系统包含文件的包含文件
// 或项目特定的包含文件。

// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef SEND_FILE_HEADER_H_
#define SEND_FILE_HEADER_H_

#include "base_task.h"

enum  eSendFileStatus
{
	eSuccess,
	eFileCreateFail,
	eChownError,
	eChmodError,
	eCreateDirFailed,
	eInvalidFilePath=10,
	eFileAlreadyExist,
	eEmptyContent,
	eInvalidContent,
	eInvalidContentType,
	eInvalidFileType,
	eInvalidSignature,
	eInalidFileMode,
	eInalidGID,
	eInalidUID,
};

bool doSendFile(const task_engine::SendFile& sendFile);

#endif 
