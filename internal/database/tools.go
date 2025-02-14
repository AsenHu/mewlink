package database

// 小工具

func chatID2Bytes(chatID int64) []byte {
	return []byte{
		byte(chatID),
		byte(chatID >> 8),
		byte(chatID >> 16),
		byte(chatID >> 24),
		byte(chatID >> 32),
		byte(chatID >> 40),
		byte(chatID >> 48),
		byte(chatID >> 56),
	}
}

func bytes2ChatID(b []byte) int64 {
	return int64(b[0]) |
		int64(b[1])<<8 |
		int64(b[2])<<16 |
		int64(b[3])<<24 |
		int64(b[4])<<32 |
		int64(b[5])<<40 |
		int64(b[6])<<48 |
		int64(b[7])<<56
}
