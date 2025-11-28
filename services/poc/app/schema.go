// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package app

const dgraphSchema = `
type Journey {
	journey.id: string
	journey.title: string
	journey.created_at: datetime
	journey.content: [uid]
}

type Content {
	content.id: string
	content.type: string
	content.minio_key: string
	content.title: string
	content.description: string
	content.uploaded_at: datetime
	content.file_size: int
	content.mime_type: string
}

journey.id: string @index(exact) .
journey.title: string .
journey.created_at: datetime .
journey.content: [uid] @reverse .
content.id: string @index(exact) .
content.type: string @index(exact) .
content.minio_key: string .
content.title: string .
content.description: string .
content.uploaded_at: datetime .
content.file_size: int .
content.mime_type: string .
`
