package constant

type MimeType string

const (
	OctetStream MimeType = "application/octet-stream"
	Wasm        MimeType = "application/wasm"

	Png  MimeType = "image/png"
	Jpeg MimeType = "image/jpeg"
	Jpg  MimeType = "image/jpg"

	HTML         MimeType = "text/html"
	CSS          MimeType = "text/css"
	JSON         MimeType = "application/json"
	ManifestJSON MimeType = "application/manifest+json"
	XML          MimeType = "application/xml"
	XHTML        MimeType = "application/xhtml+xml"
	XJavascript  MimeType = "application/x-javascript"

	ApplicationJavascript MimeType = "application/javascript"
	TextJavascript        MimeType = "text/javascript"

	Svg MimeType = "image/svg+xml"
)
