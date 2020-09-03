#ifndef _CSTRINGUTIL_H
#define _CSTRINGUTIL_H
#include <string>
#include <algorithm>

#ifdef _WIN32
#include <Windows.h>
#include <tchar.h>
#endif
class CStringUtil
{
public:
	static void ToLower(string & strBuffer)
	{
		std::transform(strBuffer.begin(),strBuffer.end(),strBuffer.begin(),::tolower);
	}
#ifdef _WIN32
	static wstring MultiByteToWideString(const std::string& szStr, unsigned int codepage)
	{
		wstring wszStr;
		int nLength = MultiByteToWideChar(codepage, 0, szStr.data(), -1, NULL, NULL);
		wszStr.resize(nLength);
		LPWSTR lpwszStr = new wchar_t[nLength];
		MultiByteToWideChar(codepage, 0, szStr.data(), -1, lpwszStr, nLength);
		wszStr = lpwszStr;
		delete[] lpwszStr;
		return wszStr;
	}

	static string WideStringToMultiByte(const wstring& wszStr, unsigned int codepage)
	{
		string szStr;
		int nLength = WideCharToMultiByte(codepage, 0, wszStr.data(), -1, NULL, 0, NULL, NULL);
		szStr.resize(nLength);
		char *lpszStr = new char[nLength];
		WideCharToMultiByte(codepage, 0, wszStr.data(), -1, lpszStr, nLength, NULL, NULL);
		szStr = lpszStr;
		delete[] lpszStr;
		return szStr;
	}

	static wstring AsciiToWideString(const string& szStr)
	{
		return MultiByteToWideString(szStr, CP_ACP);
	}

	static string WideStringToAscii(const wstring& wszStr)
	{
		return WideStringToMultiByte(wszStr, CP_ACP);
	}

	static wstring Utf8ToWideString(const string & strUtf8)
	{
		return MultiByteToWideString(strUtf8, CP_UTF8);
	}

	static string WideStringToUtf8(const wstring & wStr)
	{
		return WideStringToMultiByte(wStr, CP_UTF8);
	}

	static string AsciiToUtf8(const string & strAscii)
	{
		return WideStringToUtf8(AsciiToWideString(strAscii));
	}

	static string Utf8ToAscii(const string & strUtf8)
	{
		return WideStringToAscii(Utf8ToWideString(strUtf8));
	}
#endif
};

#endif /* _CSTRINGUTIL_H */
