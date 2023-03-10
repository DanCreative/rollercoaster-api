package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Coaster struct {
	Name         string `json:"name,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	ID           string `json:"id,omitempty"`
	InPark       string `json:"in_park,omitempty"`
	Height       int    `json:"height,omitempty"`
}

type coasterHandlers struct {
	store map[string]Coaster
	sync.Mutex
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("required env var ADMIN_PASSWORD not set")
	}

	return &adminPortal{password: password}
}

func (a adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()

	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("username or password is incorrect"))
		return
	}

	w.Write([]byte("<html><h1>Super Secret Admin Portal</h1></html>"))
}

func (h *coasterHandlers) coasters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
	case "POST":
		h.post(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("method not allowed"))
		return
	}
}

func (h *coasterHandlers) get(w http.ResponseWriter, r *http.Request) {
	coasters := make([]Coaster, len(h.store))
	i := 0
	h.Lock()
	for _, coaster := range h.store {
		coasters[i] = coaster
		i++
	}

	jsonBytes, err := json.Marshal(coasters)
	h.Unlock()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (h *coasterHandlers) getRandomCoaster(w http.ResponseWriter, r *http.Request) {
	h.Lock()
	defer h.Unlock()

	ids := make([]string, len(h.store))

	i := 0
	for id := range h.store {
		ids[i] = id
		i++
	}
	var target string
	if len(ids) == 0 {
		w.Write([]byte("No rollercoasters in the list"))
		w.WriteHeader(http.StatusNotFound)
		return
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}

	w.Header().Add("location", fmt.Sprintf("/coasters/%s", target))
	w.WriteHeader(http.StatusFound)
}

func (h *coasterHandlers) getCoaster(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.String(), "/")
	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if parts[2] == "random" {
		h.getRandomCoaster(w, r)
		return
	}

	h.Lock()
	coaster, ok := h.store[parts[2]]

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	coaster.Manufacturer = ""
	jsonBytes, err := json.Marshal(coaster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)

	h.Unlock()
}

func (h *coasterHandlers) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	var postCoaster Coaster
	err = json.Unmarshal(bodyBytes, &postCoaster)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")

	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("Need content-type 'application/json' but got '%s'", ct)))
	}

	postCoaster.ID = fmt.Sprintf("%d", time.Now().UnixNano())

	h.Lock()
	h.store[postCoaster.ID] = postCoaster
	defer h.Unlock()
}

func newCoasterHandlers() *coasterHandlers {
	return &coasterHandlers{
		store: map[string]Coaster{},
	}
}

func main() {
	coasterHandlers := newCoasterHandlers()
	admin := newAdminPortal()
	http.HandleFunc("/coasters", coasterHandlers.coasters)
	http.HandleFunc("/coasters/", coasterHandlers.getCoaster)
	http.HandleFunc("/admin", admin.handler)
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
