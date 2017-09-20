// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "http_request.h"

#include <string>

#include "json11/json11.h"
#include "CheckNet.h"
#include "Log.h"
#include "curl/curl.h"

using namespace std;
using namespace json11;

bool HttpRequest::FindRegion(std::string& host) {
  string resp;
  string url = "http://100.100.100.200/latest/meta-data/region-id";
  Log::Info("Check IP :" + url);
  bool status = http_request_get(url.c_str(), host);
  Log::Info("Check IP %d, host:%s", status, host.c_str());
  return status;
}

bool HttpRequest::DetectHost(const std::string& host) {
  string resp;
  string url = "http://" + host + "/luban/api/connection_detect";
  Log::Info("Check IP :" + url);
  bool status = http_request_post(url.c_str(), "", resp);
  Log::Info("Check IP %d", status);
  return status;
}

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
    /* Now specify the POST data */
    if(is_post) {
      curl_easy_setopt(curl, CURLOPT_POSTFIELDS, post_content.c_str());
    }

    struct curl_slist *headers = NULL;
    headers = curl_slist_append(headers, "Content-Type: application/json");

    /* pass our list of custom made headers */
    curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);

    /* send all data to this function  */
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteMemoryCallback);

    /* we pass our 'chunk' struct to the callback function */
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)&chunk);

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

    /* we pass our 'chunk' struct to the callback function */
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, (void *)fp);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1);
    /* Perform the request, res will get the return code */
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

