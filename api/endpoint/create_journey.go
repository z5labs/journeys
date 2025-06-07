package endpoint

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/z5labs/humus"
	"github.com/z5labs/humus/rest"
	"github.com/z5labs/humus/rest/rpc"

	"github.com/google/uuid"
)

type CreateJourneyHandler struct {
	log *slog.Logger
}

func CreateJourney(api *rest.Api) {
	h := &CreateJourneyHandler{
		log: humus.Logger("endpoint"),
	}

	err := api.Route(
		http.MethodPost,
		"/v1/journey",
		rpc.NewOperation(
			rpc.ConsumeJson(
				rpc.ReturnJson(h),
			),
		),
	)
	if err != nil {
		panic(err)
	}
}

type CreateJourneyRequest struct {
	Name            string    `json:"name"`
	TrackingEnabled bool      `json:"tracking_enabled"`
	StartDate       time.Time `json:"start_date"`
	EndDate         time.Time `json:"end_date"`
}

type CreateJourneyResponse struct {
	Id string `json:"id"`
}

func (h *CreateJourneyHandler) Handle(ctx context.Context, req *CreateJourneyRequest) (*CreateJourneyResponse, error) {
	h.log.InfoContext(ctx, "Handling Request", slog.String("Name", req.Name))
	resp := &CreateJourneyResponse{
		Id: uuid.New().String(),
	}
	return resp, nil
}
