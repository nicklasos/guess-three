package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	failuresCookieName = "guess_three_failures"
	failuresCookieMaxAge = 30 * 24 * 3600 // 30 days
)

type Config struct {
	A       string
	A_OTHER string
	B       string
	B_OTHER string
	C       string
	C_OTHER string
	Link    string
}

type pageData struct {
	EmojiJSON   template.JS
	Failures    int
	Locked      bool
	StatusText  string
	BootJSON    template.JS
}

type emojiPayload struct {
	A []string `json:"a"`
	B []string `json:"b"`
	C []string `json:"c"`
}

type guessResponse struct {
	OK       bool   `json:"ok"`
	Redirect string `json:"redirect,omitempty"`
	Failures int    `json:"failures,omitempty"`
	Locked   bool   `json:"locked,omitempty"`
	Error    string `json:"error,omitempty"`
}

type bootPayload struct {
	Failures int  `json:"failures"`
	Locked   bool `json:"locked"`
}

func parseCommaList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func readFailures(r *http.Request) int {
	c, err := r.Cookie(failuresCookieName)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(c.Value))
	if err != nil || n < 0 {
		return 0
	}
	if n > 3 {
		return 3
	}
	return n
}

func setFailuresCookie(w http.ResponseWriter, n int) {
	http.SetCookie(w, &http.Cookie{
		Name:     failuresCookieName,
		Value:    strconv.Itoa(n),
		Path:     "/",
		MaxAge:   failuresCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearFailuresCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     failuresCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func statusText(failures int, locked bool) string {
	if locked || failures >= 3 {
		return "Ви не відгадали"
	}
	switch failures {
	case 1:
		return "У вас лишилось 2 спроби"
	case 2:
		return "У вас лишилась остання спроба"
	default:
		return ""
	}
}

func marshalBoot(p bootPayload) template.JS {
	b, err := json.Marshal(p)
	if err != nil {
		return template.JS("{}")
	}
	return template.JS(b)
}

func writeGuessJSON(w http.ResponseWriter, status int, gr guessResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(gr); err != nil {
		log.Printf("encode guess response: %v", err)
	}
}

func handleGuess(w http.ResponseWriter, r *http.Request, answerA, answerB, answerC, winLink string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	failures := readFailures(r)
	if failures >= 3 {
		writeGuessJSON(w, http.StatusForbidden, guessResponse{OK: false, Locked: true, Failures: 3})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err != nil {
		writeGuessJSON(w, http.StatusBadRequest, guessResponse{OK: false, Error: "не вдалося прочитати запит"})
		return
	}

	var picks []string
	if err := json.Unmarshal(body, &picks); err != nil {
		writeGuessJSON(w, http.StatusBadRequest, guessResponse{OK: false, Error: "некоректний формат"})
		return
	}
	if len(picks) != 3 {
		writeGuessJSON(w, http.StatusBadRequest, guessResponse{OK: false, Error: "потрібно обрати три емодзі"})
		return
	}
	for i, p := range picks {
		picks[i] = strings.TrimSpace(p)
		if picks[i] == "" {
			writeGuessJSON(w, http.StatusBadRequest, guessResponse{OK: false, Error: "пусте емодзі"})
			return
		}
	}

	if picks[0] == answerA && picks[1] == answerB && picks[2] == answerC {
		clearFailuresCookie(w)
		writeGuessJSON(w, http.StatusOK, guessResponse{OK: true, Redirect: winLink})
		return
	}

	failures++
	setFailuresCookie(w, failures)
	resp := guessResponse{OK: false, Failures: failures}
	if failures >= 3 {
		resp.Locked = true
		writeGuessJSON(w, http.StatusOK, resp)
		return
	}
	writeGuessJSON(w, http.StatusOK, resp)
}

func main() {
	_ = godotenv.Load()

	cfg := Config{
		A:       strings.TrimSpace(os.Getenv("A")),
		A_OTHER: os.Getenv("A_OTHER"),
		B:       strings.TrimSpace(os.Getenv("B")),
		B_OTHER: os.Getenv("B_OTHER"),
		C:       strings.TrimSpace(os.Getenv("C")),
		C_OTHER: os.Getenv("C_OTHER"),
		Link:    strings.TrimSpace(os.Getenv("LINK")),
	}

	if cfg.A == "" || cfg.B == "" || cfg.C == "" || cfg.Link == "" || cfg.A_OTHER == "" || cfg.B_OTHER == "" || cfg.C_OTHER == "" {
		log.Fatal("Missing environment variables")
	}

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}
	if addr[0] != ':' {
		addr = ":" + addr
	}

	tpl := template.Must(template.ParseFiles("index.html"))

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.svg")
	})

	ansA, ansB, ansC, link := cfg.A, cfg.B, cfg.C, cfg.Link
	http.HandleFunc("/guess", func(w http.ResponseWriter, r *http.Request) {
		handleGuess(w, r, ansA, ansB, ansC, link)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		failures := readFailures(r)
		locked := failures >= 3

		payload := emojiPayload{
			A: parseCommaList(cfg.A_OTHER),
			B: parseCommaList(cfg.B_OTHER),
			C: parseCommaList(cfg.C_OTHER),
		}
		b, err := json.Marshal(payload)
		if err != nil {
			log.Printf("marshal emoji: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		boot := bootPayload{Failures: failures, Locked: locked}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := pageData{
			EmojiJSON:  template.JS(b),
			Failures:   failures,
			Locked:     locked,
			StatusText: statusText(failures, locked),
			BootJSON:   marshalBoot(boot),
		}
		if err := tpl.Execute(w, data); err != nil {
			log.Printf("template: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})

	log.Printf("serving index.html on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
