// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#define WIN32_LEAN_AND_MEAN
#include "libwebsockets.h"
//#include "private-libwebsockets.h"
#include "wskt_notifer.h"
#include <string.h>
#include <string>
#include <thread>
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "math.h"
#include "utils/MutexLocker.h"

#ifdef _WIN32
#include <windows.h>
#endif

#include "utils/Log.h"
#include "utils/host_finder.h"

#include <cstring>

#define MAX_PAYLOAD_SIZE  10 * 1024
 
thread_local int ws_status = 0;

struct session_data {
    int msg_count;
    unsigned char buf[LWS_PRE + MAX_PAYLOAD_SIZE];
    int len;
};

//using easywsclient::WebSocket;
function<void(char*)>    wskt_callback;

WsktNotifer::WsktNotifer() {
	m_path   = nullptr;
#if defined(_WIN32)
	m_worker = nullptr;
#endif
	m_stop   = false;
};


bool WsktNotifer::is_stopped() 
{
  MutexLocker(&m_mutex)
  {
    return  m_stop;
  }
}

void WsktNotifer::set_stop()
{
  MutexLocker(&m_mutex)
  {
    m_stop = true ;
  }
}

bool WsktNotifer::init(function<void(const char*)> callback) {

	m_stop     = false;
#if defined(_WIN32)
	m_worker   = new std::thread([this]() {
		poll((void*) this);
	});
#else
  pthread_create(&m_worker, nullptr, poll, (void*) this);
#endif

	return true;
};


int callback( struct lws *wsi, enum lws_callback_reasons reason, void *user, void *in, size_t len ) {
    struct session_data *data = (struct session_data *) user;
    switch ( reason ) {
        case LWS_CALLBACK_CLIENT_ESTABLISHED:   // 连接到服务器后的回调
            Log::Info("LWS_CALLBACK_CLIENT_ESTABLISHED");
            break;

        case LWS_CALLBACK_CLIENT_CONNECTION_ERROR:
            Log::Error("LWS_CALLBACK_CLIENT_CONNECTION_ERROR");
            ws_status = 1;
            break;

        case LWS_CALLBACK_CLOSED:
            Log::Error("LWS_CALLBACK_CLOSED");
            ws_status = 2;
            break;
 
        case LWS_CALLBACK_CLIENT_RECEIVE:       // 接收到服务器数据后的回调，数据为in，其长度为len
            Log::Info("data:%s", (char *) in);
            // lwsl_notice( "Rx: %s\n", (char *) in );
            if(strcmp((char *)in, "kick_vm") == 0)
              wskt_callback((char*)"kick_vm");

            break;
        case LWS_CALLBACK_CLIENT_WRITEABLE:     // 当此客户端可以发送数据时的回调
            break;

    }
    return 0;
}
 
/**
 * 支持的WebSocket子协议数组
 * 子协议即JavaScript客户端WebSocket(url, protocols)第2参数数组的元素
 * 你需要为每种协议提供回调函数
 */
struct lws_protocols protocols[] = {
    {
        //协议名称，协议回调，接收缓冲区大小
        "ws", callback, sizeof( struct session_data ), MAX_PAYLOAD_SIZE,
    },
    {
        NULL, NULL,   0 // 最后一个元素固定为此格式
    }
};

void* WsktNotifer::poll(void* args) {
	WsktNotifer* pthis = (WsktNotifer*) args;

	if ( HostFinder::getServerHost().empty() ) {
		return nullptr;
	}

	std::string host    = HostFinder::getServerHost();
	//string adrress = "ws://" + host + "/luban/notify_server";


  while ( !pthis->is_stopped() ) {
//===========================================================================
    struct lws_context_creation_info ctx_info = { 0 };
    ctx_info.port = CONTEXT_PORT_NO_LISTEN;
    ctx_info.iface = NULL;
    ctx_info.protocols = protocols;
    ctx_info.ws_ping_pong_interval = 300;
    ctx_info.gid = -1;
    ctx_info.uid = -1;
    
    AssistPath path_service("");
    std::string CfgFile = path_service.GetConfigPath() + FileUtils::separator() + "GlobalSignRootCA.crt";
      //ssl支持（指定CA证书、客户端证书及私钥路径，打开ssl支持）
    ctx_info.ssl_ca_filepath = CfgFile.c_str();//"../ca/ca-cert.pem";
    ctx_info.ssl_cert_filepath = NULL; 
    ctx_info.ssl_private_key_filepath = NULL; 
    ctx_info.options |= LWS_SERVER_OPTION_DO_SSL_GLOBAL_INIT;
 
      // 创建一个WebSocket处理器
    struct lws_context *context = lws_create_context( &ctx_info );
 
   // char address[] = (char*)host.c_str();
    int port = 443;
    char addr_port[256] = { 0 };
    sprintf(addr_port, "%s:%u", host.c_str(), port & 65535 );
 
    // 客户端连接参数
    struct lws_client_connect_info conn_info = { 0 };
    conn_info.context = context;
    conn_info.address = host.c_str();
    conn_info.port = port;
    conn_info.ssl_connection = 1;
    conn_info.path = "/luban/notify_server";
    conn_info.host = addr_port;
    conn_info.origin = addr_port;
    conn_info.protocol = protocols[ 0 ].name;
//===========================================================================
    ws_status = 0;
    struct lws *wsi = lws_client_connect_via_info( &conn_info );
      

    while (ws_status == 0 && !pthis->is_stopped() ) {
      lws_service( context, 2000 );
    }

    Log::Info("connect loop quit");
    lws_context_destroy( context );
    std::this_thread::sleep_for( std::chrono::seconds(3) );
  }

};

void WsktNotifer::unit() {
  set_stop();
#if defined(_WIN32)
	m_worker->join();
  delete m_worker;
#else
  pthread_join(m_worker, nullptr);
#endif
}




