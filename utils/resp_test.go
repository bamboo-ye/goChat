package utils

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestRespOKUsesStableJSONFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	RespOK(recorder, "value", "ok")

	var response map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["code"] != float64(0) || response["message"] != "ok" || response["data"] != "value" {
		t.Fatalf("unexpected response: %#v", response)
	}
}
