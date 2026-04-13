package ilinkapi

func unixNano() int64 {
	return timeNow().UnixNano()
}
