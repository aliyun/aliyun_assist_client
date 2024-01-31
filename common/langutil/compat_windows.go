package langutil

func LocalToUTF8(local string) string {
	if GetDefaultLang() != 0x409 {
		tmp, _ := GbkToUtf8([]byte(local))
		data := string(tmp)
		return data
	}

	return local
}

func UTF8ToLocal(utf8String string) string {
	if GetDefaultLang() != 0x409 {
		tmp, _ := Utf8ToGbk([]byte(utf8String))
		return string(tmp)
	}

	return utf8String
}
