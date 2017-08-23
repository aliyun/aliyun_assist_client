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

#ifndef SERVICE_XS_H_
#define SERVICE_XS_H_

#ifdef __cplusplus
#if __cplusplus
extern "C" {
#endif
#endif /* __cplusplus */

#include <windows.h>
#include <basetyps.h>
#include <stdlib.h>
#include <wtypes.h>
#include <initguid.h>
#include <stdio.h>
#include <string.h>
#include <winioctl.h>
#include <setupapi.h>
#include <ctype.h>
#include <strsafe.h>

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

#define XS_MAX_BUFFER 5120

  extern char* get_xen_interface_path();
  extern int xb_add_watch(HANDLE handle, char *path);
  extern int xb_wait_event(HANDLE handle);
  extern char* xb_read(HANDLE handle, char *path);
  extern int xb_write(HANDLE handle, char *path, char* info, size_t infoLen);

#ifdef __cplusplus
#if __cplusplus
}
#endif
#endif /* __cplusplus */

#endif  // SERVICE_XS_H_
