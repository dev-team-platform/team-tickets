package objectstorage

// ContentType represents a MIME type for an object.
type ContentType string

// Common content types accepted by Upload. Other MIME types can also be used.
const (
	ContentTypePDF         ContentType = "application/pdf"
	ContentTypeWord        ContentType = "application/msword"
	ContentTypeWordX       ContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	ContentTypeExcel       ContentType = "application/vnd.ms-excel"
	ContentTypeExcelX      ContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	ContentTypePowerPoint  ContentType = "application/vnd.ms-powerpoint"
	ContentTypePowerPointX ContentType = "application/vnd.openxmlformats-officedocument.presentationml.presentation"

	ContentTypePlainText ContentType = "text/plain"
	ContentTypeCSV       ContentType = "text/csv"
	ContentTypeJSON      ContentType = "application/json"
	ContentTypeXML       ContentType = "application/xml"
	ContentTypeHTML      ContentType = "text/html"

	ContentTypeJPEG ContentType = "image/jpeg"
	ContentTypePNG  ContentType = "image/png"
	ContentTypeGIF  ContentType = "image/gif"
	ContentTypeWebP ContentType = "image/webp"
	ContentTypeSVG  ContentType = "image/svg+xml"
	ContentTypeAVIF ContentType = "image/avif"
	ContentTypeICO  ContentType = "image/x-icon"

	ContentTypeZIP      ContentType = "application/zip"
	ContentTypeGzip     ContentType = "application/gzip"
	ContentTypeSevenZip ContentType = "application/x-7z-compressed"
	ContentTypeRAR      ContentType = "application/vnd.rar"

	ContentTypeMP3 ContentType = "audio/mpeg"
	ContentTypeWAV ContentType = "audio/wav"
	ContentTypeOGG ContentType = "audio/ogg"
	ContentTypeAAC ContentType = "audio/aac"

	ContentTypeMP4  ContentType = "video/mp4"
	ContentTypeWebM ContentType = "video/webm"
	ContentTypeMOV  ContentType = "video/quicktime"

	ContentTypeBinary ContentType = "application/octet-stream"
)
