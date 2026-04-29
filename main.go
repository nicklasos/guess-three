package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
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
	EmojiJSON template.JS
}

type emojiPayload struct {
	A []string `json:"a"`
	B []string `json:"b"`
	C []string `json:"c"`
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

func main() {
	_ = godotenv.Load()

	cfg := Config{
		A:       os.Getenv("A"),
		A_OTHER: os.Getenv("A_OTHER"),
		B:       os.Getenv("B"),
		B_OTHER: os.Getenv("B_OTHER"),
		C:       os.Getenv("C"),
		C_OTHER: os.Getenv("C_OTHER"),
		Link:    os.Getenv("LINK"),
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

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

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.Execute(w, pageData{EmojiJSON: template.JS(b)}); err != nil {
			log.Printf("template: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
	})

	log.Printf("serving index.html on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
