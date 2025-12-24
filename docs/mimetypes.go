package docs

type Mimetype string
type Filetype string

const (
	Image    Filetype = "image"
	Video    Filetype = "video"
	Document Filetype = "document"
	Text     Filetype = "text"
	Other    Filetype = "other"
	Illegal  Filetype = "illegal"

	MimeImageJPEG Mimetype = "image/jpeg"

	MimeImagePNG  Mimetype = "image/png"
	MimeImageGIF  Mimetype = "image/gif"
	MimeImageWebP Mimetype = "image/webp"
	MimeImageBMP  Mimetype = "image/bmp"
	MimeImageSVG  Mimetype = "image/svg+xml"
	MimeImageTIFF Mimetype = "image/tiff"

	MimeVideoMP4       Mimetype = "video/mp4"
	MimeVideoMPEG      Mimetype = "video/mpeg"
	MimeVideoQuicktime Mimetype = "video/quicktime"
	MimeVideoAVI       Mimetype = "video/x-msvideo"
	MimeVideoWebM      Mimetype = "video/webm"
	MimeVideoMKV       Mimetype = "video/x-matroska"

	MimeAudioMP3  Mimetype = "audio/mpeg"
	MimeAudioM4A  Mimetype = "audio/mp4"
	MimeAudioAAC  Mimetype = "audio/aac"
	MimeAudioOGG  Mimetype = "audio/ogg"
	MimeAudioWAV  Mimetype = "audio/wav"
	MimeAudioFLAC Mimetype = "audio/flac"

	MimeAppZIP  Mimetype = "application/zip"
	MimeAppRAR  Mimetype = "application/x-rar-compressed"
	MimeApp7Z   Mimetype = "application/x-7z-compressed"
	MimeAppTAR  Mimetype = "application/x-tar"
	MimeAppGZIP Mimetype = "application/gzip"

	MimeAppPDF  Mimetype = "application/pdf"
	MimeAppDoc  Mimetype = "application/msword"
	MimeAppDocx Mimetype = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	MimeAppXls  Mimetype = "application/vnd.ms-excel"
	MimeAppXlsx Mimetype = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	MimeAppPpt  Mimetype = "application/vnd.ms-powerpoint"
	MimeAppPptx Mimetype = "application/vnd.openxmlformats-officedocument.presentationml.presentation"

	MimeTextPlain Mimetype = "text/plain"
	MimeTextHTML  Mimetype = "text/html"
	MimeTextCSV   Mimetype = "text/csv"
	MimeTextXML   Mimetype = "text/xml"
	MimeAppJSON   Mimetype = "application/json"

	MimeTextJavaScript Mimetype = "text/javascript"
	MimeTextCSS        Mimetype = "text/css"
	MimeAppJavaScript  Mimetype = "application/javascript"

	MimeAppOctetStream Mimetype = "application/octet-stream"
	MimeAppEPUB        Mimetype = "application/epub+zip"
	MimeAppRTF         Mimetype = "application/rtf"
	MimeAppApk         Mimetype = "application/vnd.android.package-archive"
)

func GetFileCategory(mimeType Mimetype) Filetype {
	switch mimeType {
	case MimeImageJPEG, MimeImagePNG, MimeImageGIF, MimeImageWebP,
		MimeImageBMP, MimeImageSVG, MimeImageTIFF:
		return Image

	case MimeVideoMP4, MimeVideoMPEG, MimeVideoQuicktime,
		MimeVideoAVI, MimeVideoWebM, MimeVideoMKV:
		return Video

	case MimeAppPDF, MimeAppDoc, MimeAppDocx, MimeAppXls,
		MimeAppXlsx, MimeAppPpt, MimeAppPptx:
		return Document

	case MimeTextPlain, MimeTextHTML, MimeTextCSV, MimeTextXML, MimeAppJSON:
		return Text

	case MimeAppJavaScript, MimeTextJavaScript:
		return Illegal

	default:
		return Other
	}
}

func IsValidMimeType(mimeType string) bool {
	switch Mimetype(mimeType) {
	case MimeImageJPEG, MimeImagePNG, MimeImageGIF, MimeImageWebP, MimeImageBMP,
		MimeImageSVG, MimeImageTIFF, MimeVideoMP4, MimeVideoMPEG, MimeVideoQuicktime,
		MimeVideoAVI, MimeVideoWebM, MimeVideoMKV, MimeAudioMP3, MimeAudioM4A,
		MimeAudioAAC, MimeAudioOGG, MimeAudioWAV, MimeAudioFLAC, MimeAppZIP,
		MimeAppRAR, MimeApp7Z, MimeAppTAR, MimeAppGZIP, MimeAppPDF,
		MimeAppDoc, MimeAppDocx, MimeAppXls, MimeAppXlsx, MimeAppPpt, MimeAppPptx,
		MimeTextPlain, MimeTextHTML, MimeTextCSV, MimeTextXML, MimeAppJSON,
		MimeTextJavaScript, MimeTextCSS, MimeAppJavaScript,
		MimeAppOctetStream, MimeAppEPUB,
		MimeAppRTF, MimeAppApk:
		return true
	default:
		return false
	}
}
