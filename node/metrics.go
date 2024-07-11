package node

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/gorilla/mux"
	"github.com/polymerdao/monomer/environment"
	"net"
	"net/http"
	"strings"
)

func (n *Node) registerMetrics(ctx context.Context, env *environment.Env) error {
	metricsHandler := func(w http.ResponseWriter, r *http.Request) {
		format := strings.TrimSpace(r.FormValue("format"))

		gr, err := n.metrics.Gather(format)
		if err != nil {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("failed to gather metrics: %s", err))
			return
		}

		w.Header().Set("Content-Type", gr.ContentType)
		_, _ = w.Write(gr.Metrics)
	}

	metricsRouter := mux.NewRouter()
	metricsRouter.HandleFunc("/metrics", metricsHandler).Methods("GET")

	// TODO: metricsListener should be created along with the other listeners and should be passed into Node
	// 26660 is the default Prometheus port. Do we want to include the other prometheus metrics here too?
	//metricsListener, err := net.Listen("tcp", "127.0.0.1:26660")
	metricsListener, err := net.Listen("tcp", "127.0.0.1:8892")
	if err != nil {
		return fmt.Errorf("listen metrics: %v", err)
	}

	metricsService := makeHTTPService(metricsRouter, metricsListener)

	env.Go(func() {
		if err := metricsService.Run(ctx); err != nil {
		}
	})

	return nil
}

// errorResponse defines the attributes of a JSON error response.
type errorResponse struct {
	Code  int    `json:"code,omitempty"`
	Error string `json:"error"`
}

// newErrorResponse creates a new errorResponse instance.
func newErrorResponse(code int, err string) errorResponse {
	return errorResponse{Code: code, Error: err}
}

// writeErrorResponse prepares and writes a HTTP error
// given a status code and an error message.
func writeErrorResponse(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(legacy.Cdc.MustMarshalJSON(newErrorResponse(0, err)))
}
