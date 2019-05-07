/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .cpp
Description: Provide functions to get encode convert
**************************************************************************/

#ifdef _WIN32
#include <Windows.h>
#else
#include <iconv.h>
#endif // _WIN32

#define OUTLEN 1024

#include "encoder.h"


/*
UTF8 TO GBK
*/
string Encoder::Utf2Gbk(const string & utf8) {

  string stroutgbk = "";
#ifdef _WIN32
  int nwLen = MultiByteToWideChar(CP_UTF8, 0, utf8.c_str(), -1, NULL, 0);

  wchar_t * pwBuf = new wchar_t[nwLen + 1];//一定要加1，不然会出现尾巴
  memset(pwBuf, 0, nwLen * 2 + 2);

  MultiByteToWideChar(CP_UTF8, 0, utf8.c_str(), utf8.length(), pwBuf, nwLen);

  int nLen = WideCharToMultiByte(CP_ACP, 0, pwBuf, -1, NULL, NULL, NULL, NULL);

  char * pBuf = new char[nLen + 1];
  memset(pBuf, 0, nLen + 1);

  WideCharToMultiByte(CP_ACP, 0, pwBuf, nwLen, pBuf, nLen, NULL, NULL);

  stroutgbk = pBuf;
  delete[] pBuf;
  delete[] pwBuf;
  pBuf = NULL;
  pwBuf = NULL;

#else
  char out[OUTLEN];
  int rec = CodeConvert("utf-8", "gb2312", (char*)utf8.c_str(), utf8.length(), out, OUTLEN);
  stroutgbk = out;

#endif // _WIN32

  return stroutgbk;

}

/*
GBK TO UTF8
*/
string Encoder::Gbk2Utf(const string& gbk) {
  string stroututf8 = "";
#ifdef _WIN32
  WCHAR * str1;
  int n = MultiByteToWideChar(CP_ACP, 0, gbk.c_str(), -1, NULL, 0);
  str1 = new WCHAR[n];
  MultiByteToWideChar(CP_ACP, 0, gbk.c_str(), -1, str1, n);
  n = WideCharToMultiByte(CP_UTF8, 0, str1, -1, NULL, 0, NULL, NULL);
  char * str2 = new char[n];
  WideCharToMultiByte(CP_UTF8, 0, str1, -1, str2, n, NULL, NULL);
  stroututf8 = str2;
  delete[]str1;
  str1 = NULL;
  delete[]str2;
  str2 = NULL;

#else
  char out[OUTLEN];
  int rec = CodeConvert("gb2312", "utf-8", (char*)gbk.c_str(), gbk.length(), out, OUTLEN);
  stroututf8 = out;

#endif // _WIN32

  return stroututf8;

}

#ifndef _WIN32
//代码转换:从一种编码转为另一种编码
int Encoder::CodeConvert(char *from_charset, char *to_charset, char *inbuf, int inlen, char *outbuf, int outlen) {
  iconv_t cd;
  int rc;
  char **pin = &inbuf;
  char **pout = &outbuf;

  cd = iconv_open(to_charset, from_charset);
  if (cd == 0) return -1;
  memset(outbuf, 0, outlen);
  if (iconv(cd, pin, (size_t*)&inlen, pout, (size_t*)&outlen) == -1) return -1;
  iconv_close(cd);
  return 0;
}
#endif // !_WIN32

/*
BASE64 CODE
*/
static const char b64_table[] = {
  'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H',
  'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P',
  'Q', 'R', 'S', 'T', 'U', 'V', 'W', 'X',
  'Y', 'Z', 'a', 'b', 'c', 'd', 'e', 'f',
  'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n',
  'o', 'p', 'q', 'r', 's', 't', 'u', 'v',
  'w', 'x', 'y', 'z', '0', '1', '2', '3',
  '4', '5', '6', '7', '8', '9', '+', '/'
};

char* Encoder::B64Encode(const unsigned char *src, size_t len) {
  int i = 0;
  int j = 0;
  char *enc = NULL;
  size_t size = 0;
  unsigned char buf[4];
  unsigned char tmp[3];

  // alloc
  enc = (char *)malloc(0);
  if (NULL == enc) {
    return NULL;
  }

  // parse until end of source
  while (len--) {
    // read up to 3 bytes at a time into `tmp'
    tmp[i++] = *(src++);

    // if 3 bytes read then encode into `buf'
    if (3 == i) {
      buf[0] = (tmp[0] & 0xfc) >> 2;
      buf[1] = ((tmp[0] & 0x03) << 4) + ((tmp[1] & 0xf0) >> 4);
      buf[2] = ((tmp[1] & 0x0f) << 2) + ((tmp[2] & 0xc0) >> 6);
      buf[3] = tmp[2] & 0x3f;

      // allocate 4 new byts for `enc` and
      // then translate each encoded buffer
      // part by index from the base 64 index table
      // into `enc' unsigned char array
      enc = (char *)realloc(enc, size + 4);
      for (i = 0; i < 4; ++i) {
        enc[size++] = b64_table[buf[i]];
      }

      // reset index
      i = 0;
    }
  }

  // remainder
  if (i > 0) {
    // fill `tmp' with `\0' at most 3 times
    for (j = i; j < 3; ++j) {
      tmp[j] = '\0';
    }

    // perform same codec as above
    buf[0] = (tmp[0] & 0xfc) >> 2;
    buf[1] = ((tmp[0] & 0x03) << 4) + ((tmp[1] & 0xf0) >> 4);
    buf[2] = ((tmp[1] & 0x0f) << 2) + ((tmp[2] & 0xc0) >> 6);
    buf[3] = tmp[2] & 0x3f;

    // perform same write to `enc` with new allocation
    for (j = 0; (j < i + 1); ++j) {
      enc = (char *)realloc(enc, size + 1);
      enc[size++] = b64_table[buf[j]];
    }

    // while there is still a remainder
    // append `=' to `enc'
    while ((i++ < 3)) {
      enc = (char *)realloc(enc, size + 1);
      enc[size++] = '=';
    }
  }

  // Make sure we have enough space to add '\0' character at end.
  enc = (char *)realloc(enc, size + 1);
  enc[size] = '\0';

  return enc;
}

unsigned char* Encoder::B64Decode(const char *src, size_t len) {
  return B64DecodeEx(src, len, NULL);
}

unsigned char* Encoder::B64DecodeEx(const char *src, size_t len, size_t *decsize) {
  int i = 0;
  int j = 0;
  int l = 0;
  size_t size = 0;
  unsigned char *dec = NULL;
  unsigned char buf[3];
  unsigned char tmp[4];

  // alloc
  dec = (unsigned char *)malloc(0);
  if (NULL == dec) {
    return NULL;
  }

  // parse until end of source
  while (len--) {
    // break if char is `=' or not base64 char
    if ('=' == src[j]) {
      break;
    }
    if (!(isalnum(src[j]) || '+' == src[j] || '/' == src[j])) {
      break;
    }

    // read up to 4 bytes at a time into `tmp'
    tmp[i++] = src[j++];

    // if 4 bytes read then decode into `buf'
    if (4 == i) {
      // translate values in `tmp' from table
      for (i = 0; i < 4; ++i) {
        // find translation char in `b64_table'
        for (l = 0; l < 64; ++l) {
          if (tmp[i] == b64_table[l]) {
            tmp[i] = l;
            break;
          }
        }
      }

      // decode
      buf[0] = (tmp[0] << 2) + ((tmp[1] & 0x30) >> 4);
      buf[1] = ((tmp[1] & 0xf) << 4) + ((tmp[2] & 0x3c) >> 2);
      buf[2] = ((tmp[2] & 0x3) << 6) + tmp[3];

      // write decoded buffer to `dec'
      dec = (unsigned char *)realloc(dec, size + 3);
      for (i = 0; i < 3; ++i) {
        dec[size++] = buf[i];
      }

      // reset
      i = 0;
    }
  }

  // remainder
  if (i > 0) {
    // fill `tmp' with `\0' at most 4 times
    for (j = i; j < 4; ++j) {
      tmp[j] = '\0';
    }

    // translate remainder
    for (j = 0; j < 4; ++j) {
      // find translation char in `b64_table'
      for (l = 0; l < 64; ++l) {
        if (tmp[j] == b64_table[l]) {
          tmp[j] = l;
          break;
        }
      }
    }

    // decode remainder
    buf[0] = (tmp[0] << 2) + ((tmp[1] & 0x30) >> 4);
    buf[1] = ((tmp[1] & 0xf) << 4) + ((tmp[2] & 0x3c) >> 2);
    buf[2] = ((tmp[2] & 0x3) << 6) + tmp[3];

    // write remainer decoded buffer to `dec'
    dec = (unsigned char *)realloc(dec, size + (i - 1));
    for (j = 0; (j < i - 1); ++j) {
      dec[size++] = buf[j];
    }
  }

  // Make sure we have enough space to add '\0' character at end.
  dec = (unsigned char *)realloc(dec, size + 1);
  dec[size] = '\0';

  // Return back the size of decoded string if demanded.
  if (decsize != NULL) *decsize = size;

  return dec;
}


/*
const char *base64char = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

char* Encoder::Base64Encode(char *binData, char *base64, int binLength) {
    int i = 0;
    int j = 0;
    int current = 0;
    for (i = 0; i < binLength; i += 3) {

        //获取第一个6位
        current = (*(binData + i) >> 2) & 0x3F;
        *(base64 + j++) = base64char[current];

        //获取第二个6位的前两位
        current = (*(binData + i) << 4) & 0x30;

        //如果只有一个字符，那么需要做特殊处理
        if (binLength <= (i + 1)) {
            *(base64 + j++) = base64char[current];
            *(base64 + j++) = '=';
            *(base64 + j++) = '=';
            break;
        }

        //获取第二个6位的后四位
        current |= (*(binData + i + 1) >> 4) & 0xf;
        *(base64 + j++) = base64char[current];

        //获取第三个6位的前四位
        current = (*(binData + i + 1) << 2) & 0x3c;
        if (binLength <= (i + 2)) {
            *(base64 + j++) = base64char[current];
            *(base64 + j++) = '=';
            break;
        }

        //获取第三个6位的后两位
        current |= (*(binData + i + 2) >> 6) & 0x03;
        *(base64 + j++) = base64char[current];

        //获取第四个6位
        current = *(binData + i + 2) & 0x3F;
        *(base64 + j++) = base64char[current];
    }
    *(base64 + j) = '\0';

    return base64;
}

char* Encoder::Base64Decode(char const *base64Str, char *debase64Str, int encodeStrLen)
{
    int i = 0;
    int j = 0;
    int k = 0;
    char temp[4] = "";

    for (i = 0; i < encodeStrLen; i += 4) {
        for (j = 0; j < 64; j++) {
            if (*(base64Str + i) == base64char[j]) {
                temp[0] = j;
            }
        }

        for (j = 0; j < 64; j++) {
            if (*(base64Str + i + 1) == base64char[j]) {
                temp[1] = j;
            }
        }

        for (j = 0; j < 64; j++) {
            if (*(base64Str + i + 2) == base64char[j]) {
                temp[2] = j;
            }
        }

        for (j = 0; j < 64; j++) {
            if (*(base64Str + i + 3) == base64char[j]) {
                temp[3] = j;
            }
        }

        *(debase64Str + k++) = ((temp[0] << 2) & 0xFC) | ((temp[1] >> 4) & 0x03);
        if (*(base64Str + i + 2) == '=')
            break;

        *(debase64Str + k++) = ((temp[1] << 4) & 0xF0) | ((temp[2] >> 2) & 0x0F);
        if (*(base64Str + i + 3) == '=')
            break;

        *(debase64Str + k++) = ((temp[2] << 6) & 0xF0) | (temp[3] & 0x3F);
    }
    return debase64Str;
}
*/
