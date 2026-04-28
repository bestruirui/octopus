package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/model"
)

func TestDecodeDBDumpReaderSupportsWrappedDump(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{"code":200,"message":"success","data":{"version":1,"settings":[{"key":"stats_save_interval","value":"10"}]}}`

	var dump model.DBDump
	if err := decodeDBDumpReader(strings.NewReader(body), &dump); err != nil {
		t.Fatalf("decodeDBDumpReader() error = %v", err)
	}

	if dump.Version != 1 {
		t.Fatalf("dump.Version = %d, want 1", dump.Version)
	}
	if len(dump.Settings) != 1 {
		t.Fatalf("len(dump.Settings) = %d, want 1", len(dump.Settings))
	}
	if dump.Settings[0].Key != model.SettingKeyStatsSaveInterval {
		t.Fatalf("dump.Settings[0].Key = %q, want %q", dump.Settings[0].Key, model.SettingKeyStatsSaveInterval)
	}
	if dump.Settings[0].Value != "10" {
		t.Fatalf("dump.Settings[0].Value = %q, want %q", dump.Settings[0].Value, "10")
	}
}

func TestImportDBRejectsOversizedJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restore := setDBImportLimitForTest(128)
	defer restore()

	body := `{"padding":"` + strings.Repeat("a", 256) + `"}`

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/setting/import", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	importDB(c)

	assertOversizedImportResponse(t, recorder)
}

func TestImportDBRejectsOversizedMultipartFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	restore := setDBImportLimitsForTest(128, 0)
	defer restore()

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	fileWriter, err := writer.CreateFormFile("file", "backup.json")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := fileWriter.Write([]byte(`{"padding":"` + strings.Repeat("a", 256) + `"}`)); err != nil {
		t.Fatalf("fileWriter.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/setting/import", bytes.NewReader(payload.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	importDB(c)

	assertOversizedImportResponse(t, recorder)
}

func TestReadDBDumpRejectsMultipartEnvelopeLargerThanImportLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fileBody := []byte(`{"version":1}`)
	padding := strings.Repeat("a", 256)

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	if err := writer.WriteField("padding", padding); err != nil {
		t.Fatalf("WriteField() error = %v", err)
	}
	fileWriter, err := writer.CreateFormFile("file", "backup.json")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := fileWriter.Write(fileBody); err != nil {
		t.Fatalf("fileWriter.Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}

	restore := setDBImportLimitsForTest(int64(len(fileBody)+32), 64)
	defer restore()

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/setting/import", bytes.NewReader(payload.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())

	var dump model.DBDump
	err = readDBDump(c, &dump)
	if !isDBImportTooLarge(err) {
		t.Fatalf("readDBDump() error = %v, want import-size error", err)
	}
}

func setDBImportLimitForTest(limit int64) func() {
	return setDBImportLimitsForTest(limit, maxDBImportMultipartExtraBytes)
}

func setDBImportLimitsForTest(limit int64, extra int64) func() {
	original := maxDBImportBytes
	originalExtra := maxDBImportMultipartExtraBytes
	maxDBImportBytes = limit
	maxDBImportMultipartExtraBytes = extra
	return func() {
		maxDBImportBytes = original
		maxDBImportMultipartExtraBytes = originalExtra
	}
}

func assertOversizedImportResponse(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusRequestEntityTooLarge, recorder.Body.String())
	}

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("response code = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if !strings.Contains(response.Message, "backup file exceeds") {
		t.Fatalf("response message = %q, want substring %q", response.Message, "backup file exceeds")
	}
}
