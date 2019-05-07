// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef _assist_http_request_
#define _assist_http_request_

#include <string>

using namespace std;

class HttpRequest {
 public:
  static bool http_request_post(const std::string& url,
                                const std::string& post_content, std::string& response);
  static bool http_request_get(const std::string& url, std::string& response);
  static bool https_request_post(const std::string& url,
                                const std::string& post_content, std::string& response);
  static bool https_request_get(const std::string& url, std::string& response);
  static bool download_file(const std::string& url,
                            const std::string& file_path);
  HttpRequest();
 private:
  static bool https_request(const std::string& url,
      const std::string& post_content, std::string& response, bool is_post);
  static bool http_request(const std::string& url,
      const std::string& post_content, std::string& response, bool is_post);
//  static int handler(void* data, int len, uint64_t total, void* contex);
};
#endif