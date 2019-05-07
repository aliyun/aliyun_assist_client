// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "xen_notifer.h"
#include <windows.h>
#include <initguid.h>
#include <string.h>
#include <winioctl.h>
#include <setupapi.h>
#include <strsafe.h>
#include <string>
#include <thread>
#include <windows.h>
#include <tchar.h>
#include "utils/Log.h"

#define XS_PATH_CMDSTATEIN   "control/shell/statein"
#define XS_PATH_CMDSTATEOUT  "control/shell/stateout"
#define XS_PATH_CMDSTDIN     "control/shell/stdin"
#define XS_PATH_CMDSTDOUT    "control/shell/stdout"

#define LENGTH_TIMESTAMP 15
#define CMD_MAX_LENGTH   850
#define BUFFER_SIZE      850
#define STATE_ENABLE     "1"
#define EMPTY_TIMESTAMP  "00000000000000:"

/*Error message*/
#define ERR_CMD_IS_EMPTY        "51: cmd line is empty\n"
#define ERR_CMD_LAST_IS_RUNNING "52: last cmd is still running\n"
#define ERR_CMD_NOT_SUPPORT     "command is not supported\n"
#define SUC_KICK_VM             "\"result\":8, execute kick_vm success\n"
#define SHELL_CMD_TERM_PROCESS  "reset"

DEFINE_GUID(GUID_XENBUS_IFACE, 0x14ce175a, 0x3ee2, 0x4fae, 0x92, 0x52, 0x0, 0xdb, 0xd8, 0x4f, 0x1, 0x8e);
enum xsd_sockmsg_type {
	XS_DEBUG,
	XS_DIRECTORY,
	XS_READ,
	XS_GET_PERMS,
	XS_WATCH,
	XS_UNWATCH,
	XS_TRANSACTION_START,
	XS_TRANSACTION_END,
	XS_INTRODUCE,
	XS_RELEASE,
	XS_GET_DOMAIN_PATH,
	XS_WRITE,
	XS_MKDIR,
	XS_RM,
	XS_SET_PERMS,
	XS_WATCH_EVENT,
	XS_ERROR,
	XS_IS_DOMAIN_INTRODUCED,
	XS_RESUME,
	XS_SET_TARGET
};

struct xsd_sockmsg {
	ULONG type;  /* XS_??? */
	ULONG req_id;/* Request identifier, echoed in daemon's response.  */
	ULONG tx_id; /* Transaction id (0 if not related to a transaction). */
	ULONG len;   /* Length of data following this. */
				/* Generally followed by nul-terminated string(s). */
};


XenNotifer::XenNotifer() {
	m_path   = nullptr;
	m_stop   = false;
};



#define XS_MAX_BUFFER 5120
char * XenNotifer::get_xen_interface_path() {
	HDEVINFO handle;
	SP_DEVICE_INTERFACE_DATA sdid;
	SP_DEVICE_INTERFACE_DETAIL_DATA_A *sdidd;
	DWORD buf_len;
	char *path;

	handle = SetupDiGetClassDevsA(&GUID_XENBUS_IFACE, 0,
		NULL, DIGCF_PRESENT | DIGCF_DEVICEINTERFACE);
	if (handle == INVALID_HANDLE_VALUE) {
		return NULL;
	}
	sdid.cbSize = sizeof(sdid);
	if (!SetupDiEnumDeviceInterfaces(handle, NULL, &GUID_XENBUS_IFACE, 0, &sdid)) {
		Log::Error("SetupDiEnumDeviceInterfaces failed: %d", GetLastError());
		return NULL;
	}
	SetupDiGetDeviceInterfaceDetailA(handle, &sdid, NULL, 0, &buf_len, NULL);
	sdidd = (SP_DEVICE_INTERFACE_DETAIL_DATA_A*)malloc(buf_len);
	sdidd->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA_A);
	if (!SetupDiGetDeviceInterfaceDetailA(handle, &sdid, sdidd, buf_len, NULL, NULL)) {
		Log::Error("SetupDiGetDeviceInterfaceDetail failed: %d", GetLastError());
		return NULL;
	}

	path = (char*)malloc(strlen(sdidd->DevicePath) + 1);
	StringCbCopyA(path, strlen(sdidd->DevicePath) + 1, sdidd->DevicePath);
	free(sdidd);

	return path;
}

int XenNotifer::xb_add_watch(HANDLE handle, char *path) {
	char buf[XS_MAX_BUFFER];
	struct xsd_sockmsg *msg;
	DWORD bytes_written;
	DWORD bytes_read;
	char *token = "0";

	Log::Debug("add_watch start");
	msg = (struct xsd_sockmsg *)buf;
	msg->type = XS_WATCH;
	msg->req_id = 0;
	msg->tx_id = 0;
	msg->len = (ULONG)(strlen(path) + 1 + strlen(token) + 1);
	StringCbCopyA(buf + sizeof(*msg), XS_MAX_BUFFER - sizeof(*msg), path);
	StringCbCopyA(buf + sizeof(*msg) + strlen(path) + 1, XS_MAX_BUFFER - sizeof(*msg) - strlen(path) - 1, token);

	if (!WriteFile(handle, buf, sizeof(*msg) + msg->len, &bytes_written, NULL)) {
		Log::Error("WriteFile failed: %d", GetLastError());
		return 0;
	}
	if (!ReadFile(handle, buf, XS_MAX_BUFFER, &bytes_read, NULL)) {
		Log::Error("ReadFile failed: %d", GetLastError());
		return 0;
	}
	Log::Debug("bytes_read = %d", bytes_read);
	Log::Debug("msg->len = %d", msg->len);
	buf[sizeof(*msg) + msg->len] = 0;
	Log::Debug("msg text = %s", buf + sizeof(*msg));
	Log::Debug("add_watch succ end");

	return 1;
}

int XenNotifer::xb_wait_event(HANDLE handle) {
	char buf[XS_MAX_BUFFER];
	struct xsd_sockmsg *msg;
	DWORD bytes_read;

	Log::Debug("wait_event start");
	msg = (struct xsd_sockmsg *)buf;
	if (!ReadFile(handle, buf, XS_MAX_BUFFER, &bytes_read, NULL)) {
		Log::Error("ReadFile failed: %d", GetLastError());
		return 0;
	}
	Log::Debug("bytes_read = %d", bytes_read);
	Log::Debug("msg->len = %d", msg->len);
	buf[sizeof(*msg) + msg->len] = 0;
	Log::Debug("msg text = %s", buf + sizeof(*msg));
	Log::Debug("wait_event succ end");
	return 1;
}

int XenNotifer::xb_write(HANDLE handle, char *path, char* info, size_t infoLen) {
	char buf[XS_MAX_BUFFER];
	struct xsd_sockmsg *msg;
	DWORD bytes_written;
	DWORD bytes_read;
	size_t totalLen = sizeof(*msg);

	Log::Debug("write start, info : %.*s", infoLen, info);
	msg = (struct xsd_sockmsg *)buf;
	msg->type = XS_WRITE;
	msg->req_id = 0;
	msg->tx_id = 0;

	memcpy_s(buf + totalLen, XS_MAX_BUFFER - totalLen, path, strlen(path) + 1);
	totalLen += strlen(path) + 1;

	memcpy_s(buf + totalLen, XS_MAX_BUFFER - totalLen, info, infoLen);
	totalLen += infoLen;

	msg->len = (ULONG)(totalLen - sizeof(*msg));

	if (!WriteFile(handle, buf, sizeof(*msg) + msg->len, &bytes_written, NULL)) {
		Log::Error("WriteFile failed: %d", GetLastError());
		return 0;
	}
	if (!ReadFile(handle, buf, XS_MAX_BUFFER, &bytes_read, NULL)) {
		Log::Error("ReadFile failed: %d", GetLastError());
		return 0;
	}

	Log::Debug("bytes_read = %d", bytes_read);
	Log::Debug("msg->len = %d", msg->len);
	buf[sizeof(*msg) + msg->len] = 0;
	Log::Debug("msg text = %s", buf + sizeof(*msg));
	msg = (struct xsd_sockmsg *)buf;
	if (msg->type == XS_ERROR)
		return 0;
	Log::Debug("write succ end");

	return 1;
}

char * XenNotifer::xb_read(HANDLE handle, char *path) {
	char buf[XS_MAX_BUFFER];
	struct xsd_sockmsg *msg;
	char *ret;
	DWORD bytes_written;
	DWORD bytes_read;

	Log::Debug("read start");
	msg = (struct xsd_sockmsg *)buf;
	msg->type = XS_READ;
	msg->req_id = 0;
	msg->tx_id = 0;
	msg->len = (ULONG)(strlen(path) + 1);
	StringCbCopyA(buf + sizeof(*msg), XS_MAX_BUFFER - sizeof(*msg), path);

	if (!WriteFile(handle, buf, sizeof(*msg) + msg->len, &bytes_written, NULL)) {
		Log::Error("WriteFile failed: %d", GetLastError());
		return NULL;
	}

	if (!ReadFile(handle, buf, XS_MAX_BUFFER, &bytes_read, NULL)) {
		Log::Error("WriteFile failed: %d", GetLastError());
		return NULL;
	}
	Log::Debug("bytes_read = %d", bytes_read);
	Log::Debug("msg->len = %d", msg->len);
	buf[sizeof(*msg) + msg->len] = 0;
	Log::Debug("msg text = %s", buf + sizeof(*msg));
	ret = (char*)malloc(strlen(buf + sizeof(*msg)) + 1);
	StringCbCopyA(ret, XS_MAX_BUFFER - sizeof(*msg), buf + sizeof(*msg));
	Log::Debug("read succ end");
	return ret;
}



bool XenNotifer::init(function<void(const char*)> callback) {
	HANDLE hToken;
	LUID   seDebug;
	TOKEN_PRIVILEGES tkp;

	OpenProcessToken(GetCurrentProcess(),TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &hToken);
	LookupPrivilegeValue(NULL, SE_DEBUG_NAME, &seDebug);

	tkp.PrivilegeCount = 1;
	tkp.Privileges[0].Luid = seDebug;
	tkp.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED;

	AdjustTokenPrivileges(hToken, FALSE, &tkp, sizeof(tkp), NULL, NULL);
	CloseHandle(hToken);

	m_path = get_xen_interface_path();
	if ( m_path == NULL ) {
		Log::Error("XenNotifer get_xen_interface_path is NULL");
		return false;
	}

	HANDLE hFile = CreateFileA(m_path, FILE_GENERIC_READ | FILE_GENERIC_WRITE, 0,
			NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);

	if ( hFile == INVALID_HANDLE_VALUE ) {
		Log::Error("init open xen_interface_path fail");
		return false;
	}
	
	m_eventWorker = new std::thread([this]() {
		pool_shell();
	});

	m_checkWorker = new std::thread([this]() {
		pool_state();
	});

	m_shutdownWorker = new std::thread([this]() {
		pool_shutdown();
	});


	m_callback = callback;
	CloseHandle(hFile);
	return true;
};


void XenNotifer::pool_shutdown() {

	HANDLE handle = CreateFileA(m_path, FILE_GENERIC_READ | FILE_GENERIC_WRITE, 0,
		NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);

	int ret = xb_add_watch(handle, "control/shutdown");

	while ( !m_stop && xb_wait_event(handle) ) {
		char *buf = xb_read(handle, "control/shutdown");
		if (buf == NULL)
			continue;

		if (strcmp("poweroff", buf) == 0 || strcmp("halt", buf) == 0) {
			m_callback("shutdown");
		}
		else if (strcmp("reboot", buf) == 0) {
			m_callback("reboot");
		}
		else if (strcmp("hibernate", buf) == 0) {
			m_callback("hibernate");
		}

		free(buf);
	}
};

void  XenNotifer::pool_shell() {


	HANDLE watch_handle = CreateFileA(m_path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
		0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
	xb_add_watch(watch_handle, XS_PATH_CMDSTDIN);

	HANDLE hanlde = CreateFileA(m_path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
		0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);

	while ( !m_stop && xb_wait_event(watch_handle) ) {
		char* buf = xb_read(hanlde, XS_PATH_CMDSTDIN);

		if (buf == NULL)
			continue;

		Log::Info("receive event: %s", buf);

		if ( strstr(buf, "kick_vm") ) {
			m_callback("kick_vm");
			write_xenstore(hanlde, XS_PATH_CMDSTDOUT, SUC_KICK_VM,
				strlen(SUC_KICK_VM), buf);
		}
		else {
			write_xenstore(hanlde, XS_PATH_CMDSTDOUT, ERR_CMD_NOT_SUPPORT,
				strlen(ERR_CMD_NOT_SUPPORT), buf);
		}
		free(buf);
	}
	return ;
};


void  XenNotifer::pool_state() {
	HANDLE watch_handle = CreateFileA(m_path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
		0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
	xb_add_watch(watch_handle, XS_PATH_CMDSTATEIN);

	HANDLE handle = CreateFileA(m_path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
		0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
	write_xenstore(handle, XS_PATH_CMDSTATEOUT, STATE_ENABLE,
		strlen(STATE_ENABLE), NULL);

	while (!m_stop && xb_wait_event(watch_handle)) {
		write_xenstore(handle, XS_PATH_CMDSTATEOUT, STATE_ENABLE,
			strlen(STATE_ENABLE), NULL);
	}
};

void XenNotifer::unit() {
	m_stop = true;

	if (m_eventWorker) {
		m_eventWorker->join();
	}

	if (m_path){
		free(m_path);
	}
}



void XenNotifer::write_xenstore( HANDLE handle,
	char*  path,
	char*  buf,
	size_t bufLen,
	char* ptimeStamp ) {

	char writeBuf[BUFFER_SIZE + LENGTH_TIMESTAMP];
	size_t str_len;

	if (ptimeStamp != NULL) {
		if (strlen(ptimeStamp) >= LENGTH_TIMESTAMP)
			memcpy_s(writeBuf, BUFFER_SIZE + LENGTH_TIMESTAMP,
				ptimeStamp, LENGTH_TIMESTAMP);
		memcpy_s(writeBuf + LENGTH_TIMESTAMP, BUFFER_SIZE, buf, bufLen);
		str_len = bufLen + LENGTH_TIMESTAMP;
	}
	else {
		memcpy_s(writeBuf, BUFFER_SIZE + LENGTH_TIMESTAMP, buf, bufLen);
		str_len = bufLen;
	}

	Log::Info("xs_write: [%s] [%.*s] [%d]", path, str_len, writeBuf, str_len);
	xb_write(handle, path, writeBuf, str_len);
	return;
}
