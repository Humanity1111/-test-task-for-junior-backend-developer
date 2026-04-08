//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration_CreateGetListOccurrencesDelete(t *testing.T) {
	truncateTasks(t)
	base := integrationServerURL

	body := `{
		"title": "Integration daily",
		"description": "db+http",
		"status": "new",
		"scheduled_at": "2026-04-01T09:00:00Z",
		"recurrence": { "type": "daily", "interval": 1 }
	}`

	res, err := http.Post(base+"/api/v1/tasks", "application/json", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusCreated, res.StatusCode)

	var created map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&created))
	id, ok := created["id"].(float64)
	require.True(t, ok, "id should be number")
	require.Equal(t, "Integration daily", created["title"])

	res, err = http.Get(base + "/api/v1/tasks/0")
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusBadRequest, res.StatusCode)

	res, err = http.Get(base + "/api/v1/tasks/" + strconv.FormatInt(int64(id), 10))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var got map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&got))
	require.Equal(t, created["title"], got["title"])

	res, err = http.Get(base + "/api/v1/tasks")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var list []map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&list))
	require.GreaterOrEqual(t, len(list), 1)

	ocURL := base + "/api/v1/tasks/" + strconv.FormatInt(int64(id), 10) + "/occurrences?from=2026-04-01&to=2026-04-03"
	res, err = http.Get(ocURL)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	var occ map[string]any
	require.NoError(t, json.NewDecoder(res.Body).Decode(&occ))
	oc, _ := occ["occurrences"].([]any)
	require.Len(t, oc, 3)

	reqDel, err := http.NewRequest(http.MethodDelete, base+"/api/v1/tasks/"+strconv.FormatInt(int64(id), 10), nil)
	require.NoError(t, err)
	res, err = http.DefaultClient.Do(reqDel)
	require.NoError(t, err)
	res.Body.Close()
	require.Equal(t, http.StatusNoContent, res.StatusCode)

	res, err = http.Get(base + "/api/v1/tasks/" + strconv.FormatInt(int64(id), 10))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestIntegration_GetNotFound(t *testing.T) {
	truncateTasks(t)
	res, err := http.Get(integrationServerURL + "/api/v1/tasks/999999")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusNotFound, res.StatusCode)
}

func TestIntegration_InvalidJSONUnknownField(t *testing.T) {
	truncateTasks(t)
	body := `{"title":"x","description":"y","status":"new","extra":1}`
	res, err := http.Post(integrationServerURL+"/api/v1/tasks", "application/json", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestIntegration_OccurrencesValidation(t *testing.T) {
	truncateTasks(t)
	body := `{
		"title": "Monthly",
		"description": "",
		"status": "new",
		"recurrence": { "type": "monthly", "day_of_month": 5 }
	}`
	res, err := http.Post(integrationServerURL+"/api/v1/tasks", "application/json", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	require.Equal(t, http.StatusCreated, res.StatusCode, string(b))

	var created map[string]any
	require.NoError(t, json.Unmarshal(b, &created))
	id := int64(created["id"].(float64))

	res, err = http.Get(integrationServerURL + "/api/v1/tasks/" + strconv.FormatInt(id, 10) + "/occurrences?from=2026-04-30&to=2026-04-01")
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
}

