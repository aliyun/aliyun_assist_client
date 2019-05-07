// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "wskt_notifer.h"
#include <string.h>
#include <string>
#include <thread>

#ifdef _WIN32
#include <windows.h>
#else
#endif

#include "utils/Log.h"
#include "utils/host_finder.h"
#include "easywsclient/easywsclient.h"



using easywsclient::WebSocket;

WsktNotifer::WsktNotifer() {
	m_path   = nullptr;
	m_worker = nullptr;
	m_stop   = false;
};

bool WsktNotifer::init(function<void(const char*)> callback) {
	m_callback = callback;
	m_stop     = false;
	m_worker   = new std::thread([this]() {
		poll();
	});

	return true;
};

void WsktNotifer::handle_message(const std::string & message){
	if ( message == "kick_vm" ) {
		Log::Info("WsktNotifer receive kick_vm msg");
		m_callback((char*)"kick_vm");
	}
}

void  WsktNotifer::poll() {

	
	while ( HostFinder::getServerHost().empty() ) {
		std::this_thread::sleep_for( std::chrono::seconds(1) );
	}

	string host    = HostFinder::getServerHost();
	string adrress = "ws://" + host + "/luban/notify_server";

	while ( !m_stop ) {
		WebSocket::pointer ws = WebSocket::from_url( adrress );
		if ( ws == nullptr ) {
			std::this_thread::sleep_for(std::chrono::seconds(1));
			continue;
		}
		Log::Info("WsktNotifer connected");
		time_t pingTime = time(0);
		while ( ws->getReadyState() != WebSocket::CLOSED &&
			    ws->getReadyState() != WebSocket::CLOSING && !m_stop ) {
			
			ws->poll(2000);
			ws->dispatch([this](const std::string & message) {
				handle_message(message);
			});

			if ( time(0) - pingTime > 10 * 60 * 1000 ) {
				 ws->sendPing();
				 pingTime = time(0);
			}
		}
		Log::Info("WsktNotifer disconnected");
		delete ws;
		srand( (int)time(0) );
		std::this_thread::sleep_for( std::chrono::seconds( rand() % 10 + 3) );
	}
};

void WsktNotifer::unit() {
	m_stop = true;
	m_worker->join();
}




