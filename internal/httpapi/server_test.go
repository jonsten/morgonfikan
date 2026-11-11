package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jonsten/morgonfikan/internal/menu"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	config, err := menu.Parse([]byte(`schedules:
  - valid_from: 2026-01-05
    anchor_date: 2026-01-05
    weekdays:
      monday: [Monday]
      tuesday: [Tuesday]
      wednesday: [Wednesday]
      thursday: [Thursday]
      friday: [Friday]
`), []byte("closures: []\n"))
	if err != nil {
		t.Fatal(err)
	}
	zone, err := time.LoadLocation("Europe/Stockholm")
	if err != nil {
		t.Fatal(err)
	}
	return New(config, zone)
}

func TestAPIRejectsNonReadMethods(t *testing.T) {
	server := testServer(t)
	for _, path := range []string{"/api/v1/today", "/api/v1/week"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s status = %d, want 405", path, response.Code)
		}
		if response.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("POST %s missing public CORS header", path)
		}
	}
}

func TestAPIOptionsAndHead(t *testing.T) {
	server := testServer(t)
	handler := server.Handler()

	options := httptest.NewRecorder()
	handler.ServeHTTP(options, httptest.NewRequest(http.MethodOptions, "/api/v1/today", nil))
	if options.Code != http.StatusNoContent || options.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("OPTIONS response = %d, headers %v", options.Code, options.Header())
	}

	head := httptest.NewRecorder()
	handler.ServeHTTP(head, httptest.NewRequest(http.MethodHead, "/api/v1/today", nil))
	if head.Code != http.StatusOK || head.Body.Len() != 0 {
		t.Fatalf("HEAD response = %d, body %q", head.Code, head.Body.String())
	}
}

func TestAPIDayJSONShape(t *testing.T) {
	server := testServer(t)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/today", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body struct {
		Date   string   `json:"date"`
		Served bool     `json:"served"`
		Items  []string `json:"items"`
		Note   string   `json:"note"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Date == "" || !body.Served || len(body.Items) != 1 || body.Note != "" {
		t.Fatalf("unexpected day object: %+v", body)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("CORS origin = %q, want *", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestWeekJSONContainsFiveWeekdays(t *testing.T) {
	server := testServer(t)
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/week", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var body struct {
		Days []menu.Day `json:"days"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Days) != 5 {
		t.Fatalf("week has %d days, want 5", len(body.Days))
	}
	for _, day := range body.Days {
		date, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			t.Fatal(err)
		}
		if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
			t.Errorf("week contains weekend date %s", day.Date)
		}
	}
}

func TestWeekJSONStartsWithNextWeekday(t *testing.T) {
	tests := []struct {
		today string
		want  []string
	}{
		{
			today: "2026-11-11",
			want:  []string{"2026-11-12", "2026-11-13", "2026-11-16", "2026-11-17", "2026-11-18"},
		},
		{
			today: "2026-11-13",
			want:  []string{"2026-11-16", "2026-11-17", "2026-11-18", "2026-11-19", "2026-11-20"},
		},
	}

	for _, test := range tests {
		t.Run(test.today, func(t *testing.T) {
			server := testServer(t)
			date, err := time.ParseInLocation("2006-01-02", test.today, server.zone)
			if err != nil {
				t.Fatal(err)
			}
			server.now = func() time.Time { return date.Add(12 * time.Hour) }

			response := httptest.NewRecorder()
			server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/week", nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200: %s", response.Code, response.Body.String())
			}
			var body struct {
				Days []menu.Day `json:"days"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Days) != len(test.want) {
				t.Fatalf("week has %d days, want %d", len(body.Days), len(test.want))
			}
			for i, day := range body.Days {
				if day.Date != test.want[i] {
					t.Errorf("day %d date = %s, want %s", i, day.Date, test.want[i])
				}
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	response := httptest.NewRecorder()
	testServer(t).Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("health status = %d, want 204", response.Code)
	}
}
