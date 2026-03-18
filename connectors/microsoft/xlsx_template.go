package microsoft

import "encoding/base64"

// minimalXLSXBase64 is a minimal valid .xlsx file (Office Open XML) encoded as
// base64. It contains the bare minimum structure: [Content_Types].xml, package
// relationships, a workbook with one empty worksheet ("Sheet1"), and standard
// sheet dimensions.
//
// A .xlsx file is a ZIP archive with specific XML parts. Microsoft Graph's
// PUT /me/drive/root:/{path}:/content endpoint requires actual file content,
// so we embed this ~1.5 KB template rather than pulling in a dependency.
const minimalXLSXBase64 = "" +
	"UEsDBBQAAAAIAFgYcly5mqGQAQEAADsCAAATAAAAW0NvbnRlbnRfVHlwZXNdLnhtbK1RyU7DMBC9" +
	"9yssX6vYKQeEUJIeWI7AoXzA4EwSK97kcUv79zgpi4Qo4sBpNHqrZqrtZA07YCTtXc03ouQMnfKt" +
	"dn3Nn3f3xRVnlMC1YLzDmh+R+LZZVbtjQGJZ7KjmQ0rhWkpSA1og4QO6jHQ+Wkh5jb0MoEboUV6U" +
	"5aVU3iV0qUizB29WjFW32MHeJHY3ZeTUJaIhzm5O3Dmu5hCC0QpSxuXBtd+CivcQkZULhwYdaJ0J" +
	"XJ4LmcHzGV/Sx3yiqFtkTxDTA9hMlJORrz6OL96P4nefH7r6rtMKW6/2NksEhYjQ0oCYrBHLFBa0" +
	"W/+pwsInuYzNP3f59P+oUsnl980bUEsDBBQAAAAIAFgYclxdh/QutAAAACwBAAALAAAAX3JlbHMv" +
	"LnJlbHONz78OgjAQBvCdp2hul4KDMYbCYkxYDT5ALcefUHpNWxXe3o5iHBwvd9/v8hXVMmv2ROdH" +
	"MgLyNAOGRlE7ml7ArbnsjsB8kKaVmgwKWNFDVSbFFbUMMeOH0XoWEeMFDCHYE+deDThLn5JFEzcd" +
	"uVmGOLqeW6km2SPfZ9mBu08DyoSxDcvqVoCr2xxYs1r8h6euGxWeST1mNOHHl6+LKEvXYxCwaP4i" +
	"N92JpjSiwGNHvilZvgFQSwMEFAAAAAgAWBhyXNXDBk3BAAAAKAEAAA8AAAB4bC93b3JrYm9vay54" +
	"bWyNT8uOwjAMvPMVke+QlsMKVW25ICTOu/sBoXFp1Mau7LCPvycF9c7JMxrNeKY+/sXJ/KBoYGqg" +
	"3BVgkDr2gW4NfH+dtwcwmhx5NzFhA/+ocGw39S/LeGUeTfaTNjCkNFfWajdgdLrjGSkrPUt0KVO5" +
	"WZ0FndcBMcXJ7oviw0YXCF4JlbyTwX0fOjxxd49I6RUiOLmU2+sQZoV2Y0z9fKILXIkhF3P7zwWX" +
	"edFyLz4PBiNVyEAuvgT7dNvVXtt1ZfsAUEsDBBQAAAAIAFgYclz1YAOCtwAAAC0BAAAaAAAAeGwv" +
	"X3JlbHMvd29ya2Jvb2sueG1sLnJlbHONz80KwjAMB/D7nqLk7rJ5EJF1u4iwq8wHKF32gVtbmvqx" +
	"t7d4EAcePIUk5Bf+RfWcJ3Enz6M1EvI0A0FG23Y0vYRLc9rsQXBQplWTNSRhIYaqTIozTSrEGx5G" +
	"xyIihiUMIbgDIuuBZsWpdWTiprN+ViG2vken9FX1hNss26H/NqBMhFixom4l+LrNQTSLo39423Wj" +
	"pqPVt5lM+PEFH9ZfeSAKEVW+pyDhM2J8lzyNKmAMiauU5QtQSwMEFAAAAAgAWBhyXIeT3UKHAAAA" +
	"oQAAABgAAAB4bC93b3Jrc2hlZXRzL3NoZWV0MS54bWw9zEsOwjAMBNB9TxF5T11YIISSdoM4ARzA" +
	"akxb0ThRHPG5PVEXLGdG8+zwCat5cdYlioN924FhGaNfZHJwv113JzBaSDytUdjBlxWGvrHvmJ86" +
	"MxdTAVEHcynpjKjjzIG0jYmlLo+YA5Ua84SaMpPfTmHFQ9cdMdAi0DfG2K2+UCGsOP71/gdQSwEC" +
	"FAMUAAAACABYGHJcuZqhkAEBAAA7AgAAEwAAAAAAAAAAAAAAgAEAAAAAW0NvbnRlbnRfVHlwZXNd" +
	"LnhtbFBLAQIUAxQAAAAIAFgYclxdh/QutAAAACwBAAALAAAAAAAAAAAAAACAATIBAABfcmVscy8u" +
	"cmVsc1BLAQIUAxQAAAAIAFgYclzVwwZNwQAAACgBAAAPAAAAAAAAAAAAAACAAQ8CAAB4bC93b3Jr" +
	"Ym9vay54bWxQSwECFAMUAAAACABYGHJc9WADgrcAAAAtAQAAGgAAAAAAAAAAAAAAgAH9AgAAeGwv" +
	"X3JlbHMvd29ya2Jvb2sueG1sLnJlbHNQSwECFAMUAAAACABYGHJch5PdQocAAAChAAAAGAAAAAAA" +
	"AAAAAAAAgAHsAwAAeGwvd29ya3NoZWV0cy9zaGVldDEueG1sUEsFBgAAAAAFAAUARQEAAKkEAAAA" +
	"AA=="

// minimalXLSX holds the decoded XLSX template bytes, computed once at init.
// Decoding at init means any base64 corruption is caught immediately at startup
// rather than on the first user request.
var minimalXLSX []byte //nolint:gochecknoglobals // decoded once, read-only after init

func init() {
	var err error
	minimalXLSX, err = base64.StdEncoding.DecodeString(minimalXLSXBase64)
	if err != nil {
		panic("microsoft: invalid embedded XLSX template: " + err.Error())
	}
}
