//go:build integration

package internal_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/UnitVectorY-Labs/itemservicecentral/internal/config"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/database"
	"github.com/UnitVectorY-Labs/itemservicecentral/internal/handler"

	_ "github.com/lib/pq"
)

var testServer *httptest.Server
var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = database.Connect("localhost", 5433, "testdb", "test", "test", "disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to connect to database: %v\n", err)
		os.Exit(1)
	}

	// Drop tables if they exist from previous runs
	testDB.Exec(`DROP TABLE IF EXISTS "items"`)
	testDB.Exec(`DROP TABLE IF EXISTS "orders"`)
	testDB.Exec(`DROP TABLE IF EXISTS _meta`)

	tables := testTables()

	if err := database.Migrate(testDB, tables, database.MigrateOptions{}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run migrations: %v\n", err)
		os.Exit(1)
	}

	store := database.NewStore(testDB)

	h, err := handler.New(store, tables)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create handler: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	h.SetupRoutes(mux)

	testServer = httptest.NewServer(mux)

	code := m.Run()

	testServer.Close()
	testDB.Exec(`DROP TABLE IF EXISTS "items"`)
	testDB.Exec(`DROP TABLE IF EXISTS "orders"`)
	testDB.Exec(`DROP TABLE IF EXISTS _meta`)
	testDB.Close()

	os.Exit(code)
}

func testTables() []config.TableConfig {
	return []config.TableConfig{
		{
			Name: "items",
			PrimaryKey: config.KeyConfig{
				Field:   "itemId",
				Pattern: `^[A-Za-z_][A-Za-z0-9._-]*$`,
			},
			RangeKey:       nil,
			AllowTableScan: true,
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"itemId":   map[string]interface{}{"type": "string", "pattern": `^[A-Za-z_][A-Za-z0-9._-]*$`},
					"name":     map[string]interface{}{"type": "string"},
					"status":   map[string]interface{}{"type": "string"},
					"category": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"itemId", "name"},
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_status",
					PrimaryKey: config.KeyConfig{
						Field: "status",
					},
					AllowIndexScan: true,
				},
			},
		},
		{
			Name: "orders",
			PrimaryKey: config.KeyConfig{
				Field:   "orderId",
				Pattern: `^[A-Za-z_][A-Za-z0-9._-]*$`,
			},
			RangeKey: &config.KeyConfig{
				Field:   "lineId",
				Pattern: `^[A-Za-z_][A-Za-z0-9._-]*$`,
			},
			AllowTableScan: false,
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"orderId":    map[string]interface{}{"type": "string", "pattern": `^[A-Za-z_][A-Za-z0-9._-]*$`},
					"lineId":     map[string]interface{}{"type": "string", "pattern": `^[A-Za-z_][A-Za-z0-9._-]*$`},
					"customerId": map[string]interface{}{"type": "string"},
					"amount":     map[string]interface{}{"type": "number"},
				},
				"required": []interface{}{"orderId", "lineId", "customerId"},
			},
			Indexes: []config.IndexConfig{
				{
					Name: "by_customer",
					PrimaryKey: config.KeyConfig{
						Field: "customerId",
					},
					AllowIndexScan: false,
				},
			},
		},
	}
}

// --- Helper functions ---

func putItem(t *testing.T, server *httptest.Server, path string, body map[string]interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, server.URL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	return resp
}

func getItem(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(server.URL + path)
	if err != nil {
		t.Fatalf("GET request failed: %v", err)
	}
	return resp
}

func patchItem(t *testing.T, server *httptest.Server, path string, body map[string]interface{}) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPatch, server.URL+path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH request failed: %v", err)
	}
	return resp
}

func deleteItem(t *testing.T, server *httptest.Server, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, server.URL+path, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE request failed: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatalf("failed to unmarshal response body: %v\nbody: %s", err, string(b))
	}
	return result
}

func readListBody(t *testing.T, resp *http.Response) ([]interface{}, string) {
	t.Helper()
	body := readBody(t, resp)
	items, ok := body["items"].([]interface{})
	if !ok {
		t.Fatalf("expected items array in response, got: %v", body)
	}
	nextCursor, _ := body["nextCursor"].(string)
	return items, nextCursor
}

// --- PK-only table tests (items) ---

func TestPutItem_PKOnly(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/item1/_item", map[string]interface{}{
		"itemId": "item1",
		"name":   "Widget",
		"status": "active",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	result := readBody(t, resp)
	if result["itemId"] != "item1" {
		t.Errorf("expected itemId=item1, got %v", result["itemId"])
	}
	if result["name"] != "Widget" {
		t.Errorf("expected name=Widget, got %v", result["name"])
	}

	// GET it back
	resp2 := getItem(t, testServer, "/v1/items/data/item1/_item")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", resp2.StatusCode)
	}
	result2 := readBody(t, resp2)
	if result2["itemId"] != "item1" {
		t.Errorf("expected itemId=item1, got %v", result2["itemId"])
	}
	if result2["name"] != "Widget" {
		t.Errorf("expected name=Widget, got %v", result2["name"])
	}
}

func TestPutItem_PKOnly_Update(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/itemUpdate/_item", map[string]interface{}{
		"itemId": "itemUpdate",
		"name":   "OriginalName",
		"status": "draft",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp2 := putItem(t, testServer, "/v1/items/data/itemUpdate/_item", map[string]interface{}{
		"itemId": "itemUpdate",
		"name":   "UpdatedName",
		"status": "published",
	})
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	resp3 := getItem(t, testServer, "/v1/items/data/itemUpdate/_item")
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", resp3.StatusCode)
	}
	result := readBody(t, resp3)
	if result["name"] != "UpdatedName" {
		t.Errorf("expected name=UpdatedName, got %v", result["name"])
	}
	if result["status"] != "published" {
		t.Errorf("expected status=published, got %v", result["status"])
	}
}

func TestGetItem_PKOnly_NotFound(t *testing.T) {
	resp := getItem(t, testServer, "/v1/items/data/nonexistent/_item")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestDeleteItem_PKOnly(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/itemDel/_item", map[string]interface{}{
		"itemId": "itemDel",
		"name":   "ToDelete",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	delResp := deleteItem(t, testServer, "/v1/items/data/itemDel/_item")
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}
	delResp.Body.Close()

	getResp := getItem(t, testServer, "/v1/items/data/itemDel/_item")
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getResp.StatusCode)
	}
	getResp.Body.Close()
}

func TestPatchItem_PKOnly(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/itemPatch/_item", map[string]interface{}{
		"itemId":   "itemPatch",
		"name":     "PatchMe",
		"status":   "draft",
		"category": "tools",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	patchResp := patchItem(t, testServer, "/v1/items/data/itemPatch/_item", map[string]interface{}{
		"status": "active",
	})
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on PATCH, got %d", patchResp.StatusCode)
	}
	result := readBody(t, patchResp)
	if result["status"] != "active" {
		t.Errorf("expected status=active, got %v", result["status"])
	}
	if result["name"] != "PatchMe" {
		t.Errorf("expected name=PatchMe preserved, got %v", result["name"])
	}
	if result["category"] != "tools" {
		t.Errorf("expected category=tools preserved, got %v", result["category"])
	}
}

func TestPatchItem_PKOnly_NotFound(t *testing.T) {
	resp := patchItem(t, testServer, "/v1/items/data/noSuchItem/_item", map[string]interface{}{
		"status": "active",
	})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPutItem_PKOnly_InvalidJSON(t *testing.T) {
	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/v1/items/data/itemBad/_item", bytes.NewReader([]byte("not json")))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPutItem_PKOnly_SchemaViolation(t *testing.T) {
	// Missing required "name" field
	resp := putItem(t, testServer, "/v1/items/data/itemSchema/_item", map[string]interface{}{
		"itemId": "itemSchema",
		"status": "active",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPutItem_PKOnly_InvalidKeyValue(t *testing.T) {
	// Key with space is invalid for ValidateKeyValue
	req, err := http.NewRequest(http.MethodPut, testServer.URL+"/v1/items/data/bad%20key/_item", bytes.NewReader([]byte(`{"itemId":"bad key","name":"test"}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestPutItem_PKOnly_PKMismatch(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/itemA/_item", map[string]interface{}{
		"itemId": "itemB",
		"name":   "Mismatch",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestListItems_PKOnly(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/listPK/_item", map[string]interface{}{
		"itemId": "listPK",
		"name":   "Listed",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	listResp := getItem(t, testServer, "/v1/items/data/listPK/_items")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listResp.StatusCode)
	}
	items, _ := readListBody(t, listResp)
	if len(items) < 1 {
		t.Fatalf("expected at least 1 item, got %d", len(items))
	}
	found := false
	for _, it := range items {
		m := it.(map[string]interface{})
		if m["itemId"] == "listPK" {
			found = true
		}
	}
	if !found {
		t.Error("expected to find listPK in listed items")
	}
}

func TestScanTable_PKOnly(t *testing.T) {
	// Insert multiple items
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("scan%d", i)
		resp := putItem(t, testServer, "/v1/items/data/"+id+"/_item", map[string]interface{}{
			"itemId": id,
			"name":   fmt.Sprintf("Scan Item %d", i),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	scanResp := getItem(t, testServer, "/v1/items/_items?limit=2")
	if scanResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on scan, got %d", scanResp.StatusCode)
	}
	items, nextCursor := readListBody(t, scanResp)
	if len(items) != 2 {
		t.Fatalf("expected 2 items in first page, got %d", len(items))
	}
	if nextCursor == "" {
		t.Fatal("expected a nextCursor for pagination")
	}

	// Fetch next page
	scanResp2 := getItem(t, testServer, "/v1/items/_items?limit=2&cursor="+nextCursor)
	if scanResp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on second scan page, got %d", scanResp2.StatusCode)
	}
	items2, _ := readListBody(t, scanResp2)
	if len(items2) == 0 {
		t.Fatal("expected items in second page")
	}
}

func TestQueryIndex_PKOnly(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/idxItem1/_item", map[string]interface{}{
		"itemId": "idxItem1",
		"name":   "Indexed",
		"status": "idx_active",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp2 := putItem(t, testServer, "/v1/items/data/idxItem2/_item", map[string]interface{}{
		"itemId": "idxItem2",
		"name":   "Also Indexed",
		"status": "idx_active",
	})
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	qResp := getItem(t, testServer, "/v1/items/_index/by_status/idx_active/_items")
	if qResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on index query, got %d", qResp.StatusCode)
	}
	items, _ := readListBody(t, qResp)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items from index query, got %d", len(items))
	}
}

// --- PK+RK table tests (orders) ---

func TestPutItem_PKRK(t *testing.T) {
	resp := putItem(t, testServer, "/v1/orders/data/order1/line1/_item", map[string]interface{}{
		"orderId":    "order1",
		"lineId":     "line1",
		"customerId": "cust1",
		"amount":     42.5,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	getResp := getItem(t, testServer, "/v1/orders/data/order1/line1/_item")
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on GET, got %d", getResp.StatusCode)
	}
	result := readBody(t, getResp)
	if result["orderId"] != "order1" {
		t.Errorf("expected orderId=order1, got %v", result["orderId"])
	}
	if result["lineId"] != "line1" {
		t.Errorf("expected lineId=line1, got %v", result["lineId"])
	}
	if result["customerId"] != "cust1" {
		t.Errorf("expected customerId=cust1, got %v", result["customerId"])
	}
}

func TestPutItem_PKRK_MultipleRKs(t *testing.T) {
	for i := 1; i <= 3; i++ {
		lineId := fmt.Sprintf("mline%d", i)
		resp := putItem(t, testServer, "/v1/orders/data/orderMulti/"+lineId+"/_item", map[string]interface{}{
			"orderId":    "orderMulti",
			"lineId":     lineId,
			"customerId": "custMulti",
			"amount":     float64(i * 10),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	listResp := getItem(t, testServer, "/v1/orders/data/orderMulti/_items")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listResp.StatusCode)
	}
	items, _ := readListBody(t, listResp)
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestListItems_PKRK(t *testing.T) {
	for i := 1; i <= 2; i++ {
		lineId := fmt.Sprintf("lline%d", i)
		resp := putItem(t, testServer, "/v1/orders/data/orderList/"+lineId+"/_item", map[string]interface{}{
			"orderId":    "orderList",
			"lineId":     lineId,
			"customerId": "custList",
			"amount":     float64(i * 5),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	listResp := getItem(t, testServer, "/v1/orders/data/orderList/_items")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on list, got %d", listResp.StatusCode)
	}
	items, _ := readListBody(t, listResp)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestListItems_PKRK_RKFilter(t *testing.T) {
	for _, lid := range []string{"alpha1", "alpha2", "beta1"} {
		resp := putItem(t, testServer, "/v1/orders/data/orderFilter/"+lid+"/_item", map[string]interface{}{
			"orderId":    "orderFilter",
			"lineId":     lid,
			"customerId": "custFilter",
			"amount":     1.0,
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	listResp := getItem(t, testServer, "/v1/orders/data/orderFilter/_items?rkBeginsWith=alpha")
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.StatusCode)
	}
	items, _ := readListBody(t, listResp)
	if len(items) != 2 {
		t.Fatalf("expected 2 items with rkBeginsWith=alpha, got %d", len(items))
	}
}

func TestDeleteItem_PKRK(t *testing.T) {
	resp := putItem(t, testServer, "/v1/orders/data/orderDel/lineDel/_item", map[string]interface{}{
		"orderId":    "orderDel",
		"lineId":     "lineDel",
		"customerId": "custDel",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	delResp := deleteItem(t, testServer, "/v1/orders/data/orderDel/lineDel/_item")
	if delResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delResp.StatusCode)
	}
	delResp.Body.Close()

	getResp := getItem(t, testServer, "/v1/orders/data/orderDel/lineDel/_item")
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getResp.StatusCode)
	}
	getResp.Body.Close()
}

func TestPatchItem_PKRK(t *testing.T) {
	resp := putItem(t, testServer, "/v1/orders/data/orderPatch/linePatch/_item", map[string]interface{}{
		"orderId":    "orderPatch",
		"lineId":     "linePatch",
		"customerId": "custPatch",
		"amount":     10.0,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	patchResp := patchItem(t, testServer, "/v1/orders/data/orderPatch/linePatch/_item", map[string]interface{}{
		"amount": 99.0,
	})
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on PATCH, got %d", patchResp.StatusCode)
	}
	result := readBody(t, patchResp)
	if result["customerId"] != "custPatch" {
		t.Errorf("expected customerId=custPatch preserved, got %v", result["customerId"])
	}
	amt, ok := result["amount"].(float64)
	if !ok || amt != 99.0 {
		t.Errorf("expected amount=99, got %v", result["amount"])
	}
}

func TestQueryIndex_PKRK(t *testing.T) {
	for i := 1; i <= 2; i++ {
		lineId := fmt.Sprintf("qline%d", i)
		resp := putItem(t, testServer, "/v1/orders/data/orderIdx/"+lineId+"/_item", map[string]interface{}{
			"orderId":    "orderIdx",
			"lineId":     lineId,
			"customerId": "custIdx",
			"amount":     float64(i),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	qResp := getItem(t, testServer, "/v1/orders/_index/by_customer/custIdx/_items")
	if qResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", qResp.StatusCode)
	}
	items, _ := readListBody(t, qResp)
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items from index query, got %d", len(items))
	}
}

// --- JSON validation tests ---

func TestPutItem_InvalidJSONKeys(t *testing.T) {
	resp := putItem(t, testServer, "/v1/items/data/underscoreTest/_item", map[string]interface{}{
		"itemId":     "underscoreTest",
		"name":       "Test",
		"_badField":  "value",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for underscore key, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

// --- Pagination tests ---

func TestListItems_Pagination(t *testing.T) {
	// Insert 5 items with the same prefix for pagination testing
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("page%d", i)
		resp := putItem(t, testServer, "/v1/items/data/"+id+"/_item", map[string]interface{}{
			"itemId": id,
			"name":   fmt.Sprintf("Page Item %d", i),
		})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		resp.Body.Close()
	}

	// Scan with limit=2 and paginate
	allItems := make([]interface{}, 0)
	cursor := ""
	pages := 0
	for {
		url := "/v1/items/_items?limit=2"
		if cursor != "" {
			url += "&cursor=" + cursor
		}
		resp := getItem(t, testServer, url)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		items, next := readListBody(t, resp)
		allItems = append(allItems, items...)
		pages++
		if next == "" {
			break
		}
		cursor = next
		if pages > 20 {
			t.Fatal("too many pages, possible infinite loop")
		}
	}

	if len(allItems) < 5 {
		t.Fatalf("expected at least 5 items across all pages, got %d", len(allItems))
	}
}
