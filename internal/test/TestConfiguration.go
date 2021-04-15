//  +build test

package testconfiguration

import (
	"os"
)

// Create creates a configuration for unit testing
func Create(initFiles bool) {
	os.Setenv("GOKAPI_CONFIG_DIR", "test")
	os.Setenv("GOKAPI_DATA_DIR", "test")
	os.Mkdir("test", 0777)
	os.WriteFile("test/config.json", configTestFile, 0777)
	if initFiles {
		os.Mkdir("test/data", 0777)
		os.WriteFile("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0", []byte("123"), 0777)
		os.WriteFile("test/data/c4f9375f9834b4e7f0a528cc65c055702bf5f24a", []byte("456"), 0777)
		os.WriteFile("test/data/e017693e4a04a59d0b0f400fe98177fe7ee13cf7", []byte("789"), 0777)
		os.WriteFile("test/fileupload.jpg", []byte("abc"), 0777)
	}
}

// Delete deletes the configuration for unit testing
func Delete() {
	os.RemoveAll("test")
}

var configTestFile = []byte(`{"Port":"127.0.0.1:53843","AdminName":"test","AdminPassword":"10340aece68aa4fb14507ae45b05506026f276cf","ServerUrl":"http://127.0.0.1:53843/","DefaultDownloads":3,"DefaultExpiry":20,"DefaultPassword":"123","RedirectUrl":"https://test.com/","Sessions":{"validsession":{"RenewAt":2147483645,"ValidUntil":2147483646},"expiredsession":{"RenewAt":0,"ValidUntil":0}},"Files":{"Wzol7LyY2QVczXynJtVo":{"Id":"Wzol7LyY2QVczXynJtVo","Name":"smallfile2","Size":"8 B","SHA256":"e017693e4a04a59d0b0f400fe98177fe7ee13cf7","ExpireAt":2147483646,"ExpireAtString":"2021-05-04 15:19","DownloadsRemaining":1,"PasswordHash":"","HotlinkId":""},"e4TjE7CokWK0giiLNxDL":{"Id":"e4TjE7CokWK0giiLNxDL","Name":"smallfile2","Size":"8 B","SHA256":"e017693e4a04a59d0b0f400fe98177fe7ee13cf7","ExpireAt":2147483645,"ExpireAtString":"2021-05-04 15:19","DownloadsRemaining":2,"PasswordHash":"","HotlinkId":""},"jpLXGJKigM4hjtA6T6sN":{"Id":"jpLXGJKigM4hjtA6T6sN","Name":"smallfile","Size":"7 B","SHA256":"c4f9375f9834b4e7f0a528cc65c055702bf5f24a","ExpireAt":2147483646,"ExpireAtString":"2021-05-04 15:18","DownloadsRemaining":1,"PasswordHash":"7b30508aa9b233ab4b8a11b2af5816bdb58ca3e7","HotlinkId":""},"n1tSTAGj8zan9KaT4u6p":{"Id":"n1tSTAGj8zan9KaT4u6p","Name":"picture.jpg","Size":"4 B","SHA256":"a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0","ExpireAt":2147483646,"ExpireAtString":"2021-05-04 15:19","DownloadsRemaining":1,"PasswordHash":"","HotlinkId":"PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg"}},"Hotlinks":{"PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg":{"Id":"PhSs6mFtf8O5YGlLMfNw9rYXx9XRNkzCnJZpQBi7inunv3Z4A.jpg","FileId":"n1tSTAGj8zan9KaT4u6p"}},"ConfigVersion":4,"SaltAdmin":"LW6fW4Pjv8GtdWVLSZD66gYEev6NAaXxOVBw7C","SaltFiles":"lL5wMTtnVCn5TPbpRaSe4vAQodWW0hgk00WCZE","LengthId":20,"DataDir":"test/data"}`)
