package main

import (
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/go-chi/chi/v5"
	"github.com/unrolled/render"
)

func httpError(w http.ResponseWriter, r *http.Request, error string) {
	SetFlash(w, "message", []byte(error))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type Landing struct {
	Languages []string
	Flash     string
}

func landingHandler(w http.ResponseWriter, r *http.Request) {
	flash, _ := GetFlash(w, r, "message")

	landing := Landing{
		Languages: lexers.Names(false),
		Flash:     flash,
	}

	t.HTML(w, http.StatusOK, "landing", landing)
}

func robotsHandler(w http.ResponseWriter, r *http.Request) {
	t.Text(w, http.StatusOK, "User-agent: *\nAllow: /$\nDisallow: /")
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	// Prevent paste requests to "favicon.ico"
	http.Error(w, "Not found", http.StatusNotFound)
}

func cliHandler(w http.ResponseWriter, r *http.Request) {
	bytes, _ := os.ReadFile("paste.py")
	script := string(bytes)

	t.Text(w, http.StatusOK, strings.ReplaceAll(script, "__URL__", os.Getenv("CLI_URL")))
}

func pasteHandler(w http.ResponseWriter, r *http.Request) {
	// fetch paste
	paste, err := getPaste(chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, r, "This paste does not exist or may have expired")
		return
	}

	t.HTML(w, http.StatusOK, "paste", paste)
}

func pasteRawHandler(w http.ResponseWriter, r *http.Request) {
	// fetch paste
	paste, err := getPaste(chi.URLParam(r, "id"))
	if err != nil {
		httpError(w, r, "This paste does not exist or may have expired")
		return
	}

	t.Text(w, http.StatusOK, paste.Content)
}

func createPasteHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Println(err.Error())
		httpError(w, r, "Could not parse form.")
		return
	}

	var paste Paste
	if err := decoder.Decode(&paste, r.PostForm); err != nil {
		log.Println(err.Error())
		httpError(w, r, "Could not decode form to struct.")
		return
	}

	paste.Id = randomId(6)

	err := insertPaste(paste)
	if err != nil {
		httpError(w, r, "Failed to insert paste to database. Please try again.")
		return
	}

	http.Redirect(w, r, paste.Id, http.StatusSeeOther)
}

func initTemplate() {
	t = render.New(render.Options{
		Directory:  "views/",
		Layout:     "layout",
		Extensions: []string{".html"},
		Funcs: []template.FuncMap{{
			"unescape": func(s string) template.HTML {
				return template.HTML(s)
			},
			"humanize": func(d time.Duration) string {
				return humanizeDuration(d)
			},
			"duration": func(p Paste) time.Duration {
				return durationPaste(p)
			},
		}},
	})
}

func startHttp(port int) {
	initTemplate()

	r := chi.NewRouter()
	r.Get("/", landingHandler)
	r.Get("/robots.txt", robotsHandler)
	r.Get("/favicon.ico", faviconHandler)
	r.Get("/cli", cliHandler)
	r.Get("/{id}", pasteHandler)
	r.Get("/{id}/raw", pasteRawHandler)
	r.Post("/", createPasteHandler)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))

	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Server is now listening on http://0.0.0.0:%d", port)

	if err := http.Serve(l, r); err != nil {
		log.Fatal(err)
	}
}
