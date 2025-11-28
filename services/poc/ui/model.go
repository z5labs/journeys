package ui

import "time"

type Content struct {
	ID           string
	Type         string
	MinioKey     string
	Title        string
	Description  string
	UploadedAt   time.Time
	FileSize     int64
	MimeType     string
	Latitude     *float64
	Longitude    *float64
	Altitude     *float64
	LocationName string
	CapturedAt   *time.Time
}

// ContentSection represents content grouped by a specific date
type ContentSection struct {
	DateKey     string    // "2025-01-15" or "Unknown"
	DisplayDate string    // Display text for section header
	SortOrder   int64     // 0 for Unknown, negative unix timestamp for dates
	Contents    []Content // Already sorted by time
}

// journeyViewModel is the data passed to the journey view template
type journeyViewModel struct {
	ID       string
	Title    string
	Sections []ContentSection
}

type journey struct {
	ID       string
	Title    string
	Contents []Content
}
