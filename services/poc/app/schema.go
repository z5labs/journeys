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
}

journey.id: string @index(exact) .
journey.title: string .
journey.created_at: datetime .
`
