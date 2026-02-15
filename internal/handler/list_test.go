package handler

import (
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/database"
)

func TestBuildListMetaAlwaysReturnsObject(t *testing.T) {
	meta := buildListMeta(&database.ListResult{})
	if meta.NextPageToken != "" || meta.PreviousPageToken != "" {
		t.Fatalf("expected empty list meta, got %#v", meta)
	}
}
