package repo

// DisableEncryptorForTest 临时去掉加密器（模拟没配加密密钥），返回恢复函数。只在测试里使用
func DisableEncryptorForTest() (restore func()) {
	saved := globalEncryptor
	globalEncryptor = nil
	return func() { globalEncryptor = saved }
}
