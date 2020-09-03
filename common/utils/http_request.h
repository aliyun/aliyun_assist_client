// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef _assist_http_request_
#define _assist_http_request_

#include <string>

using namespace std;

class HttpRequest {
  enum ContentType {
    json,
    text,
  };

 public:
  static bool http_request_post(const std::string& url,
                                const std::string& post_content, std::string& response);
  static bool http_request_get(const std::string& url, std::string& response);
  static bool https_request_post(const std::string& url,
                                const std::string& post_content, std::string& response);
  static bool https_request_get(const std::string& url, std::string& response);
  static bool https_request_post_text(const std::string& url,
                                      const std::string& post_content, std::string& response);
  static bool download_file(const std::string& url,
                            const std::string& file_path);
  static bool url_encode(const std::string& str, std::string& result);

  HttpRequest();
 private:
  static bool https_request(const std::string& url,
      const std::string& post_content, 
      std::string& response, 
      bool is_post, 
      ContentType content_type = ContentType::json);
  static bool http_request(const std::string& url,
      const std::string& post_content, std::string& response, bool is_post);
//  static int handler(void* data, int len, uint64_t total, void* contex);
};
#endif
