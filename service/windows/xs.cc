/* 
    Xen Store Daemon interface providing simple tree-like database.
    Copyright (C) 2005 Rusty Russell IBM Corporation

    This library is free software; you can redistribute it and/or
    modify it under the terms of the GNU Lesser General Public
    License as published by the Free Software Foundation; either
    version 2.1 of the License, or (at your option) any later version.

    This library is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
    Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public
    License along with this library; if not, write to the Free Software
    Foundation, Inc., 51 Franklin St, Fifth Floor, Boston, MA  02110-1301  USA
*/

#include "./xs.h"
#include "utils/Log.h"

char * get_xen_interface_path() {
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

int xb_add_watch(HANDLE handle, char *path) {
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

int xb_wait_event(HANDLE handle) {
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

int xb_write(HANDLE handle, char *path, char* info, size_t infoLen) {
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

char * xb_read(HANDLE handle, char *path) {
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
