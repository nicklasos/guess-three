package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	A       string
	B       string
	C       string
	D       string
	E       string
	A_OTHER string
	B_OTHER string
	C_OTHER string
	D_OTHER string
	E_OTHER string
	Link    string
}

type pageData struct {
	EmojiJSON template.JS
}

type emojiPayload struct {
	A []string `json:"a"`
	B []string `json:"b"`
	C []string `json:"c"`
	D []string `json:"d"`
	E []string `json:"e"`
}

type guessResponse struct {
	OK       bool   `json:"ok"`
	Redirect string `json:"redirect,omitempty"`
	Error    string `json:"error,omitempty"`
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

func writeGuessJSON(w http.ResponseWriter, status int, gr guessResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(gr); err != nil {
		log.Printf("encode guess response: %v", err)
	}
}

func handleGuess(w http.ResponseWriter, r *http.Request, ansA, ansB, ansC, ansD, ansE, winLink string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
	if len(picks) != 5 {
		writeGuessJSON(w, http.StatusBadRequest, guessResponse{OK: false, Error: "потрібно обрати п'ять емодзі"})
		return
	}
	for i, p := range picks {
		picks[i] = strings.TrimSpace(p)
		if picks[i] == "" {
			writeGuessJSON(w, http.StatusBadRequest, guessResponse{OK: false, Error: "пусте емодзі"})
			return
		}
	}

	if picks[0] == ansA && picks[1] == ansB && picks[2] == ansC && picks[3] == ansD && picks[4] == ansE {
		writeGuessJSON(w, http.StatusOK, guessResponse{OK: true, Redirect: winLink})
		return
	}

	writeGuessJSON(w, http.StatusOK, guessResponse{OK: false})
}

func main() {
	_ = godotenv.Load()

	cfg := Config{
		A:       strings.TrimSpace(os.Getenv("A")),
		B:       strings.TrimSpace(os.Getenv("B")),
		C:       strings.TrimSpace(os.Getenv("C")),
		D:       strings.TrimSpace(os.Getenv("D")),
		E:       strings.TrimSpace(os.Getenv("E")),
		A_OTHER: os.Getenv("A_OTHER"),
		B_OTHER: os.Getenv("B_OTHER"),
		C_OTHER: os.Getenv("C_OTHER"),
		D_OTHER: os.Getenv("D_OTHER"),
		E_OTHER: os.Getenv("E_OTHER"),
		Link:    strings.TrimSpace(os.Getenv("LINK")),
	}

	missing :=
		cfg.A == "" ||
			cfg.B == "" ||
			cfg.C == "" ||
			cfg.D == "" ||
			cfg.E == "" ||
			cfg.A_OTHER == "" ||
			cfg.B_OTHER == "" ||
			cfg.C_OTHER == "" ||
			cfg.D_OTHER == "" ||
			cfg.E_OTHER == "" ||
			cfg.Link == ""

	if missing {
		log.Fatal("Missing environment variables")
	}

	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8786"
	}
	if addr[0] != ':' {
		addr = ":" + addr
	}

	tpl := template.Must(template.ParseFiles("index.html"))

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "favicon.svg")
	})

	http.HandleFunc("/guess", func(w http.ResponseWriter, r *http.Request) {
		handleGuess(w, r, cfg.A, cfg.B, cfg.C, cfg.D, cfg.E, cfg.Link)
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

		payload := emojiPayload{
			A: parseCommaList(cfg.A_OTHER),
			B: parseCommaList(cfg.B_OTHER),
			C: parseCommaList(cfg.C_OTHER),
			D: parseCommaList(cfg.D_OTHER),
			E: parseCommaList(cfg.E_OTHER),
		}
		b, err := json.Marshal(payload)
		if err != nil {
			log.Printf("marshal emoji: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := pageData{EmojiJSON: template.JS(b)}
		if err := tpl.Execute(w, data); err != nil {
			log.Printf("template: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	})

	log.Printf("serving index.html on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
