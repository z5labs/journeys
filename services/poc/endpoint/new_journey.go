// Copyright (c) 2025 Z5labs and Contributors
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package endpoint

import (
	"context"
	"net/http"

	"github.com/dgraph-io/dgo/v240"
	"github.com/z5labs/humus/rest"
)

type newJourneyHandler struct {
	dgraph *dgo.Dgraph
}

func NewJourney(dgraph *dgo.Dgraph) rest.ApiOption {
	h := &newJourneyHandler{
		dgraph: dgraph,
	}

	return rest.Operation(
		http.MethodPost,
		rest.BasePath("/api/v1/journeys"),
		rest.ReturnJson(
			rest.ConsumeJson(h),
		),
	)
}

type NewJourneyRequest struct {
	Title string `json:"title"`
}

type NewJourneyResponse struct {
	ID string `json:"id"`
}

func (h *newJourneyHandler) Handle(ctx context.Context, req *NewJourneyRequest) (*NewJourneyResponse, error) {
	return &NewJourneyResponse{}, nil
}
