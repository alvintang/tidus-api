package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestHealthCheckHandler(t *testing.T) {
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheckHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body is what we expect.
	expected := `{"alive": true}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestDataHandler(t *testing.T) {
	tt := []struct {
		routeVariable string
		idVariable    string
		status        int
	}{
		{"info", "1", 200},
		{"info", "10", 200},
		{"info", "99", 404},
		{"code", "1", 200},
		{"code", "10", 200},
		{"code", "99", 404},
	}

	for _, tc := range tt {
		path := fmt.Sprintf("/data/%s/%s/", tc.routeVariable, tc.idVariable)
		req, err := http.NewRequest("GET", path, nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		// Need to create a router that we can pass the request through so that the vars will be added to the context
		router := mux.NewRouter()
		router.HandleFunc("/data/{category}/{id}/", InfoHandler)
		router.ServeHTTP(rr, req)

		// In this case, our MetricsHandler returns a non-200 response
		// for a route variable it doesn't know about.
		if rr.Code != tc.status {
			t.Errorf("handler should have failed on routeVariable %s: got %v want %v",
				tc.routeVariable, rr.Code, http.StatusOK)
		}

		expected := `{"message"`
		if !strings.HasPrefix(rr.Body.String(), expected) {
			t.Errorf("handler returned unexpected body: got %v want %v",
				rr.Body.String(), expected)
		}
	}
}
