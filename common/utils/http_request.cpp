// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "http_request.h"

#include <string>

#include "json11/json11.h"
#include "host_finder.h"
#include "AssistPath.h"
#include "Log.h"
#include "curl/curl.h"
#include "FileUtil.h"
#include "SystemInfo.h"

using namespace std;
using namespace json11;

#define DOWNLOAD_FILE_TIME_OUT 30

HttpRequest::HttpRequest() {
	static bool inited = false;
	if (!inited) {
		inited = true;
		curl_global_init(CURL_GLOBAL_ALL);
	}
}

HttpRequest initialize;

namespace {

struct MemoryStruct {
  std::string memory;
  size_t size;
};

static size_t WriteMemoryCallback(void *contents, size_t size, size_t nmemb, void *userp) {
  size_t realsize = size * nmemb;
  struct MemoryStruct *mem = (struct MemoryStruct *)userp;

  mem->memory.append((char *)contents, realsize);
  mem->size += realsize;

  return realsize;
}

static size_t WriteFileCallback(void *contents, size_t size, size_t nmemb, void *userp) {
  size_t realsize = size * nmemb;
  fwrite(contents, size, nmemb, (FILE *)userp);
  return realsize;
}

}

bool HttpRequest::http_request_get(const std::string& url, std::string& response) {
  return http_request(url, "", response, false);
}

bool HttpRequest::http_request_post(const std::string& url,
    const std::string& post_content, std::string& response) {
  return http_request(url, post_content, response, true);
}

bool HttpRequest::https_request_get(const std::string& url, std::string& response) {
  return https_request(url, "", response, false);
}

bool HttpRequest::https_request_post(const std::string& url,
    const std::string& post_content, std::string& response) {
  return https_request(url, post_content, response, true);
}

bool HttpRequest::https_request_post_text(const std::string& url,
  const std::string& post_content, std::string& response) {
  return https_request(url, post_content, response, true, ContentType::text);
}

bool HttpRequest::http_request(const std::string& url,
                                    const std::string& post_content, std::string& response, bool is_post) {
  CURL *curl;
  CURLcode res = CURLE_OK;

  struct MemoryStruct chunk;
  chunk.size = 0;  /* no data at this point */
  /* get a curl handle */
  curl = curl_easy_init();
  if(curl) {
    /* First set the URL that is about to receive our POST. This URL can
       just as well be a https:// URL if that is what should receive the
       data. */
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 5);
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);
	curl_easy_setopt(curl, CURLOPT_FAILONERROR, 1L);
    /* Now specify the POST data */
    if(is_post) {
      curl_easy_setopt(curl, CURLOPT_POSTFIELDS, post_content.c_str());
    }

    struct curl_slist *headers = NULL;
    headers = curl_slist_append(headers, "Content-Type: application/json; charset=utf-8");
	

    /* pass our list of custom made headers */
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);

    /* send all data to this function  */
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteMemoryCallback);

    /* we pass our 'chunk' struct to the callback function */
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);


  //  curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
  //  curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
  //  curl_easy_setopt(curl, CURLOPT_CAPATH ,"C:\download");

  //  curl_easy_setopt(curl, CURLOPT_CAINFO ,"c:\\download\\cacert.pem");
    /* Perform the request, res will get the return code */
    res = curl_easy_perform(curl);

    curl_slist_free_all(headers); /* free the header list */
    /* Check for errors */
    if(res != CURLE_OK)
      Log::Error("%s curl_easy_perform() failed: %s\n", url.c_str(), 
                 curl_easy_strerror(res));

    response = chunk.memory;
    /* always cleanup */
    curl_easy_cleanup(curl);
  }
  return res == CURLE_OK;
}


bool HttpRequest::https_request(const std::string& url,
    const std::string& post_content,
    std::string& response,
    bool is_post,
    ContentType content_type) {
  CURL *curl;
  CURLcode res = CURLE_OK;

  struct MemoryStruct chunk;
  chunk.size = 0;  /* no data at this point */
  /* get a curl handle */
  curl = curl_easy_init();
  if(curl) {
    /* First set the URL that is about to receive our POST. This URL can
       just as well be a https:// URL if that is what should receive the
       data. */
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 3);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, 5);
    curl_easy_setopt(curl, CURLOPT_NOSIGNAL, 1L);
	curl_easy_setopt(curl, CURLOPT_FAILONERROR, 1L);
    /* Now specify the POST data */
    if(is_post) {
      curl_easy_setopt(curl, CURLOPT_POSTFIELDS, post_content.c_str());
    }

    struct curl_slist *headers = NULL;

    if(content_type == ContentType::json) {
      headers = curl_slist_append(headers, "Content-Type: application/json; charset=utf-8");
    } else if(content_type == ContentType::text) {
      headers = curl_slist_append(headers, "Content-Type: text/plain; charset=utf-8");
    } else {
      Log::Error("Wrong Content-Type: %d\n", content_type);
      return false;
    }
    std::string all_ips = SystemInfo::GetAllIPs();
    if (!all_ips.empty()) {
      char client_ip[512];
      sprintf(client_ip, "X-Client-IP: %s", all_ips.c_str());
      headers = curl_slist_append(headers, client_ip);
    }

    /* pass our list of custom made headers */
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);

    /* send all data to this function  */
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteMemoryCallback);

    /* we pass our 'chunk' struct to the callback function */
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);

	//curl_easy_setopt(curl, CURLOPT_VERBOSE, 1);
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 2L);
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 1L);
    AssistPath path_service("");

    string CfgFile = path_service.GetConfigPath() + FileUtils::separator() + "GlobalSignRootCA.crt";
    curl_easy_setopt(curl, CURLOPT_CAINFO , CfgFile.c_str());
    /* Perform the request, res will get the return code */
    res = curl_easy_perform(curl);

    curl_slist_free_all(headers); /* free the header list */
    /* Check for errors */
    if(res != CURLE_OK)
      Log::Error("%s curl_easy_perform() failed: %s\n", url.c_str(), 
                 curl_easy_strerror(res));

    response = chunk.memory;
    /* always cleanup */
    curl_easy_cleanup(curl);
  }
  return res == CURLE_OK;
}

bool HttpRequest::download_file(const std::string& url,
                                const std::string& file_path) {
  CURL *curl;
  CURLcode res = CURLE_OK;
  FILE * fp = fopen(file_path.c_str(), "wb");
  if (fp == nullptr) {
    return false;
  }
  curl = curl_easy_init();
  if(curl) {
    /* First set the URL that is about to receive our POST. This URL can
       just as well be a https:// URL if that is what should receive the
       data. */
    curl_easy_setopt(curl, CURLOPT_URL, url.c_str());

    /* send all data to this function  */
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteFileCallback);

    curl_easy_setopt(curl, CURLOPT_CONNECTTIMEOUT, 10);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, DOWNLOAD_FILE_TIME_OUT);

    /* we pass our 'chunk' struct to the callback function */
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)fp);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1);
    /* Perform the request, res will get the return code */

    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYPEER, 0L);
    curl_easy_setopt(curl, CURLOPT_SSL_VERIFYHOST, 0L);
    AssistPath path_service("");

    string CfgFile = path_service.GetConfigPath() + FileUtils::separator() + "GlobalSignRootCA.crt";
    curl_easy_setopt(curl, CURLOPT_CAINFO , CfgFile.c_str());

    res = curl_easy_perform(curl);

    if(res != CURLE_OK) {
      Log::Error("curl_easy_perform() failed: %s\n",
                 curl_easy_strerror(res));
    }

    /* always cleanup */
    curl_easy_cleanup(curl);
  }
  fclose(fp);
  return res == CURLE_OK;
}

bool HttpRequest::url_encode(const std::string& str, std::string& result) {
    if (str.length() == 0) {
        return 1; // empty str encode return success
    }
    CURL * curl = curl_easy_init();
    bool success = 0;
    if (curl) {
        char * output = curl_easy_escape(curl, str.c_str(), str.length());
        if (output) {
            result = std::string(output);
            curl_free(output);
            success = 1;
        }
        curl_easy_cleanup(curl);
    }
    if (success == 0) {
        Log::Error("url_encode() failed, encode str: %s", str.c_str());
    }
    return success;
}

