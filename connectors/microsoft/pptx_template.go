package microsoft

import "encoding/base64"

// minimalPPTXBase64 is a minimal valid .pptx file (Office Open XML) encoded as
// base64. It contains the bare minimum structure: [Content_Types].xml, package
// relationships, and an empty presentation with standard slide dimensions.
//
// A .pptx file is a ZIP archive with specific XML parts. Microsoft Graph's
// PUT /me/drive/root:/{path}:/content endpoint requires actual file content,
// so we embed this ~1.2 KB template rather than pulling in a dependency.
const minimalPPTXBase64 = "" +
	"UEsDBBQAAAAIAEKIZlxNR4pI7wAAAL0BAAATAAAAW0NvbnRlbnRfVHlwZXNdLnhtbH2QzU7DMBCE" +
	"730Ka68odsoBIZSkBwpH4FAeYOVsEgv/yd5W7dvjpCB+RDmu5puZ3W02R2fFgVI2wbewljUI8jr0" +
	"xo8tvO4eqxsQmdH3aIOnFk6UYdOtmt0pUhbF7HMLE3O8UyrriRxmGSL5ogwhOeQyplFF1G84kru" +
	"u6xulg2fyXPGcAd1KiGZLA+4ti4djUc67JLIZxP2ZnetawBit0chFVwff/yqqPkpkcS5MnkzMVwUA" +
	"dalkFi93fFmfy4uS6Um8YOIndAVUMbKKiXKxLrj8P+yPhcMwGE190HtXLPJ7mLM/RunQ+M9TGrV8" +
	"v3sHUEsDBBQAAAAIAEKIZlw62VMktAAAADEBAAALAAAAX3JlbHMvLnJlbHONz80KwjAMB/D7nqLk" +
	"7rp5EBG7XUTYVeYDlDbrhusHTRX39hZPTjx4TPLPL+TYPu3MHhhp8k5AXVbA0CmvJ2cEXPvzZg+M" +
	"knRazt6hgAUJ2qY4XnCWKe/QOAViGXEkYEwpHDgnNaKVVPqALk8GH61MuYyGB6lu0iDfVtWOx08D" +
	"moKxFcs6LSB2ugbWLwH/4f0wTApPXt0tuvTjylciyzIaTAJCSDxEpNx8p8ssA8+P8tWnzQtQSwME" +
	"FAAAAAgAQohmXPlVXkjdAAAAlwEAABQAAABwcHQvcHJlc2VudGF0aW9uLnhtbI2QwU4DMQxE7/2K" +
	"yHeaLSrVstpsLwgJCU7AB0SJtxspcaI4QMvXE0pXpbcexx4/zbjf7oMXn5jZRVKwWjYgkEy0jnYK" +
	"3t8eb1oQXDRZ7SOhggMybIdFn7qUkZGKLvVSVApxpxVMpaROSjYTBs3LmJDqbow56FJl3kmb9Vel" +
	"By9vm2Yjg3YEC3Ei5GsIcRydwYdoPkIN8IfJ6I9JeHKJz7x0De9/k4tYQ+XUpuzti+aC+ck+c5Hn" +
	"6eu3MHsF96v1umnq58xBwaa9a3/FbKNYkE/GeXc0zlfV2MvLdw4/UEsDBBQAAAAIAEKIZlyMDoXQ" +
	"fQAAAJ0AAAAfAAAAcHB0L19yZWxzL3ByZXNlbnRhdGlvbi54bWwucmVsc1XMQQ7CIBCF4b2nILO3" +
	"oAtjTGl3PYDRA0zoCI0wEIYYvb0sdfny533j/E5RvajKltnCYTCgiF1eN/YW7rdlfwYlDXnFmJks" +
	"fEhgnnbjlSK2/pGwFVEdYbEQWisXrcUFSihDLsS9PHJN2PqsXhd0T/Skj8acdP01oKP6T52+UEsB" +
	"AhQDFAAAAAgAQohmXE1HikjvAAAAvQEAABMAAAAAAAAAAAAAAIABAAAAAFtDb250ZW50X1R5cGVz" +
	"XS54bWxQSwECFAMUAAAACABCiGZcOtlTJLQAAAAxAQAACwAAAAAAAAAAAAAAgAEgAQAAX3JlbHMv" +
	"LnJlbHNQSwECFAMUAAAACABCiGZc+VVeSN0AAACXAQAAFAAAAAAAAAAAAAAAgAH9AQAAcHB0L3By" +
	"ZXNlbnRhdGlvbi54bWxQSwECFAMUAAAACABCiGZcjA6F0H0AAACdAAAAHwAAAAAAAAAAAAAAgAEM" +
	"AwAAcHB0L19yZWxzL3ByZXNlbnRhdGlvbi54bWwucmVsc1BLBQYAAAAABAAEAAkBAADGAwAAAAA="

// minimalPPTX holds the decoded PPTX template bytes, computed once at init.
// Decoding at init means any base64 corruption is caught immediately at startup
// rather than on the first user request.
var minimalPPTX []byte //nolint:gochecknoglobals // decoded once, read-only after init

func init() {
	var err error
	minimalPPTX, err = base64.StdEncoding.DecodeString(minimalPPTXBase64)
	if err != nil {
		panic("microsoft: invalid embedded PPTX template: " + err.Error())
	}
}
