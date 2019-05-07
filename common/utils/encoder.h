/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .h
Description: Provide functions to get encode convert
**************************************************************************/

#ifndef PROJECT_ENCODE_H_
#define PROJECT_ENCODE_H_

#include <string.h>
#include <stdio.h>
#include <iostream>
#include <stdlib.h>

using  std::string;

class Encoder {
 public:
  static string Utf2Gbk(const string & utf8);
  static string Gbk2Utf(const string & utf8);

  //char * Base64Encode(char * binData, char * base64, int binLength);
  //char * Base64Decode(char const * base64Str, char * debase64Str, int encodeStrLen);

  char * B64Encode(const unsigned char * src, size_t len);
  unsigned char * B64Decode(const char * src, size_t len);
  unsigned char * B64DecodeEx(const char * src, size_t len, size_t * decsize);

 private:
  static int CodeConvert(char *from_charset, char *to_charset, char *inbuf, int inlen, char *outbuf, int outlen);

};

#endif //PROJECT_ENCODE_H_

