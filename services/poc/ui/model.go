package ui

import "time"

type Content struct {
	ID          string
	Type        string
	MinioKey    string
	Title       string
	Description string
	UploadedAt  time.Time
	FileSize    int64
	MimeType    string
}

type journey struct {
	ID       string
	Title    string
	Contents []Content
}
