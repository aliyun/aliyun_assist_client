package langutil

func LocalToUTF8(local string) string {
	if GetDefaultLang() != 0x409 {
		tmp, _ := GbkToUtf8([]byte(local))
		data := string(tmp)
		return data
	}

	return local
}
