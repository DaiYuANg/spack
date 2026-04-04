package constant

type MimeType string

const (
	OctetStream MimeType = "application/octet-stream"

	Png  MimeType = "image/png"
	Jpeg MimeType = "image/jpeg"
	Jpg  MimeType = "image/jpg"

	HTML MimeType = "text/html"
	CSS  MimeType = "text/css"
	JSON MimeType = "application/json"

	ApplicationJavascript MimeType = "application/javascript"
	TextJavascript        MimeType = "text/javascript"

	Svg MimeType = "image/svg+xml"
)
