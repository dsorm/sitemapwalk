package app

func IsGzip(b []byte) bool {
	if len(b) < 2 {
		return false
	}

	return b[0] == 0x1f && b[1] == 0x8b
}
