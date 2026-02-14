package handler

import (
	"fmt"
	"net/http"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/database"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/schema"
)

// Handler is the top-level HTTP handler that dispatches to per-table handlers.
type Handler struct {
	store  *database.Store
	tables map[string]*tableHandler
}

// tableHandler holds the configuration and compiled schema for a single table.
type tableHandler struct {
	config    config.TableConfig
	validator *schema.Validator
	indexes   map[string]config.IndexConfig
}

// New creates a Handler by compiling schemas and building index lookup maps.
func New(store *database.Store, tables []config.TableConfig) (*Handler, error) {
	h := &Handler{
		store:  store,
		tables: make(map[string]*tableHandler, len(tables)),
	}

	for _, t := range tables {
		v, err := schema.Compile(t.Schema)
		if err != nil {
			return nil, fmt.Errorf("table %q: failed to compile schema: %w", t.Name, err)
		}

		idxMap := make(map[string]config.IndexConfig, len(t.Indexes))
		for _, idx := range t.Indexes {
			idxMap[idx.Name] = idx
		}

		h.tables[t.Name] = &tableHandler{
			config:    t,
			validator: v,
			indexes:   idxMap,
		}
	}

	return h, nil
}

// SetupRoutes registers all API routes on the given ServeMux.
func (h *Handler) SetupRoutes(mux *http.ServeMux) {
	for name, th := range h.tables {
		h.registerTableRoutes(mux, name, th)
	}
}

func (h *Handler) registerTableRoutes(mux *http.ServeMux, name string, th *tableHandler) {
	hasRK := th.config.RangeKey != nil

	if hasRK {
		mux.HandleFunc("GET /v1/"+name+"/data/{pk}/{rk}/_item", h.handleGetItem(th))
		mux.HandleFunc("PUT /v1/"+name+"/data/{pk}/{rk}/_item", h.handlePutItem(th))
		mux.HandleFunc("PATCH /v1/"+name+"/data/{pk}/{rk}/_item", h.handlePatchItem(th))
		mux.HandleFunc("DELETE /v1/"+name+"/data/{pk}/{rk}/_item", h.handleDeleteItem(th))
	} else {
		mux.HandleFunc("GET /v1/"+name+"/data/{pk}/_item", h.handleGetItem(th))
		mux.HandleFunc("PUT /v1/"+name+"/data/{pk}/_item", h.handlePutItem(th))
		mux.HandleFunc("PATCH /v1/"+name+"/data/{pk}/_item", h.handlePatchItem(th))
		mux.HandleFunc("DELETE /v1/"+name+"/data/{pk}/_item", h.handleDeleteItem(th))
	}

	// List items within a partition
	mux.HandleFunc("GET /v1/"+name+"/data/{pk}/_items", h.handleListItems(th))

	// Table scan
	if th.config.AllowTableScan {
		mux.HandleFunc("GET /v1/"+name+"/_items", h.handleScanTable(th))
	}

	// Index routes
	for _, idx := range th.config.Indexes {
		// Query index by pk
		mux.HandleFunc("GET /v1/"+name+"/_index/"+idx.Name+"/{indexPk}/_items", h.handleQueryIndex(th, idx))

		// Index scan
		if idx.AllowIndexScan {
			mux.HandleFunc("GET /v1/"+name+"/_index/"+idx.Name+"/_items", h.handleScanIndex(th, idx))
		}

		// Get single item by index pk+rk
		if idx.RangeKey != nil {
			mux.HandleFunc("GET /v1/"+name+"/_index/"+idx.Name+"/{indexPk}/{indexRk}/_item", h.handleGetIndexItem(th, idx))
		}
	}
}
