package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/VictoriaMetrics-Community/mcp-victoriatraces/cmd/mcp-victoriatraces/config"
)

func TestLogsQLToolHandlers(t *testing.T) {
	var gotPath string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	t.Setenv("VT_INSTANCE_ENTRYPOINT", srv.URL)
	cfg, err := config.InitConfig()
	if err != nil {
		t.Fatalf("Failed to init config: %v", err)
	}

	type handler func(context.Context, *config.Config, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	testCases := []struct {
		name          string
		handler       handler
		args          map[string]any
		expectedPath  string
		expectedQuery url.Values
	}{
		{
			name:          "query",
			handler:       toolQueryHandler,
			args:          map[string]any{"query": "*", "start": "1h"},
			expectedPath:  "/select/logsql/query",
			expectedQuery: url.Values{"query": {"*"}, "start": {"1h"}, "limit": {"1000"}},
		},
		{
			name:          "stats_query without time",
			handler:       toolStatsQueryHandler,
			args:          map[string]any{"query": "* | stats count()"},
			expectedPath:  "/select/logsql/stats_query",
			expectedQuery: url.Values{"query": {"* | stats count()"}},
		},
		{
			name:          "stats_query_range",
			handler:       toolStatsQueryRangeHandler,
			args:          map[string]any{"query": "* | stats count()", "start": "1700000000", "step": "5m"},
			expectedPath:  "/select/logsql/stats_query_range",
			expectedQuery: url.Values{"query": {"* | stats count()"}, "start": {"1700000000"}, "step": {"5m"}},
		},
		{
			name:          "hits",
			handler:       toolHitsHandler,
			args:          map[string]any{"query": "*", "start": "1700000000", "field": []any{"resource_attr:service.name", "name"}, "fields_limit": float64(5)},
			expectedPath:  "/select/logsql/hits",
			expectedQuery: url.Values{"query": {"*"}, "start": {"1700000000"}, "field": {"resource_attr:service.name", "name"}, "fields_limit": {"5"}},
		},
		{
			name:          "field_names",
			handler:       toolFieldNamesHandler,
			args:          map[string]any{"query": "*", "start": "1700000000"},
			expectedPath:  "/select/logsql/field_names",
			expectedQuery: url.Values{"query": {"*"}, "start": {"1700000000"}},
		},
		{
			name:          "field_values",
			handler:       toolFieldValuesHandler,
			args:          map[string]any{"query": "*", "start": "1700000000", "field": "name", "limit": float64(10)},
			expectedPath:  "/select/logsql/field_values",
			expectedQuery: url.Values{"query": {"*"}, "start": {"1700000000"}, "field": {"name"}, "limit": {"10"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tcr := mcp.CallToolRequest{}
			tcr.Params.Arguments = tc.args

			result, err := tc.handler(context.Background(), cfg, tcr)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result.IsError {
				t.Fatalf("Unexpected error result: %v", result.Content)
			}
			if gotPath != tc.expectedPath {
				t.Errorf("Expected path %q, got %q", tc.expectedPath, gotPath)
			}
			if gotQuery.Encode() != tc.expectedQuery.Encode() {
				t.Errorf("Expected query %q, got %q", tc.expectedQuery.Encode(), gotQuery.Encode())
			}
		})
	}
}
