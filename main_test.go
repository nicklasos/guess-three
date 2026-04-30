package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestParseCommaList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty string", in: "", want: nil},
		{name: "only commas/spaces", in: " , ,  ", want: nil},
		{name: "single item", in: "🍎", want: []string{"🍎"}},
		{name: "trim spaces", in: "  a , b  , c", want: []string{"a", "b", "c"}},
		{name: "emoji list", in: "😀,🙂,😊", want: []string{"😀", "🙂", "😊"}},
		{name: "no extra empty from trailing comma", in: "x,y,", want: []string{"x", "y"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseCommaList(tt.in)
			if tt.want == nil {
				if len(got) != 0 {
					t.Fatalf("got %v, want nil or empty", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func decodeGuessResponse(tb testing.TB, body string) guessResponse {
	tb.Helper()
	var gr guessResponse
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&gr); err != nil {
		tb.Fatal(err)
	}
	return gr
}

func TestHandleGuess(t *testing.T) {
	t.Parallel()

	const (
		a, b, c, d, e = "🚕", "🐇", "🔮", "🌃", "💡"
		winURL        = "https://example.com/win"
	)

	t.Run("method not POST", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/guess", nil)
		handleGuess(rr, req, a, b, c, d, e, winURL)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status %d", rr.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/guess", strings.NewReader("not-json"))
		handleGuess(rr, req, a, b, c, d, e, winURL)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status want 400 got %d", rr.Code)
		}
		gr := decodeGuessResponse(t, rr.Body.String())
		if gr.OK || gr.Error == "" {
			t.Fatalf("got %+v", gr)
		}
		if !strings.Contains(gr.Error, "некорект") {
			t.Fatalf("unexpected error message: %q", gr.Error)
		}
	})

	t.Run("wrong number of picks", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		body := mustJSON(t, []string{"🚕", "🐇", "🔮"})
		req := httptest.NewRequest(http.MethodPost, "/guess", strings.NewReader(body))
		handleGuess(rr, req, a, b, c, d, e, winURL)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status want 400 got %d", rr.Code)
		}
		gr := decodeGuessResponse(t, rr.Body.String())
		if gr.OK || !strings.Contains(gr.Error, "п'ять") {
			t.Fatalf("got %+v", gr)
		}
	})

	t.Run("empty pick after trim", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		body := mustJSON(t, []string{"🚕", "🐇", "🔮", "🌃", "   "})
		req := httptest.NewRequest(http.MethodPost, "/guess", strings.NewReader(body))
		handleGuess(rr, req, a, b, c, d, e, winURL)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status want 400 got %d", rr.Code)
		}
		gr := decodeGuessResponse(t, rr.Body.String())
		if gr.OK || !strings.Contains(gr.Error, "пусте") {
			t.Fatalf("got %+v", gr)
		}
	})

	t.Run("wrong combination", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		body := mustJSON(t, []string{"🚕", "🐇", "🔮", "🌃", "⭐"})
		req := httptest.NewRequest(http.MethodPost, "/guess", strings.NewReader(body))
		handleGuess(rr, req, a, b, c, d, e, winURL)

		if rr.Code != http.StatusOK {
			t.Fatalf("status want 200 got %d", rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Fatalf("Content-Type got %q", ct)
		}
		gr := decodeGuessResponse(t, rr.Body.String())
		if gr.OK || gr.Redirect != "" || gr.Error != "" {
			t.Fatalf("got %+v", gr)
		}
	})

	t.Run("correct picks trims whitespace", func(t *testing.T) {
		t.Parallel()
		rr := httptest.NewRecorder()
		body := mustJSON(t, []string{" " + a + " ", b, c, d, e})
		req := httptest.NewRequest(http.MethodPost, "/guess", strings.NewReader(body))
		handleGuess(rr, req, a, b, c, d, e, winURL)

		if rr.Code != http.StatusOK {
			t.Fatalf("status %d body %s", rr.Code, rr.Body.String())
		}
		gr := decodeGuessResponse(t, rr.Body.String())
		if !gr.OK || gr.Redirect != winURL || gr.Error != "" {
			t.Fatalf("got %+v", gr)
		}
	})
}

func mustJSON(tb testing.TB, v any) string {
	tb.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		tb.Fatal(err)
	}
	return string(b)
}
