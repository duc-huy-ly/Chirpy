package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func okResponseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) requestLogger(w http.ResponseWriter, r *http.Request) {
	hits := fmt.Sprintf(`
	<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(hits))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits = atomic.Int32{}
}

func validate(w http.ResponseWriter, r *http.Request) {
	const maxChirpSize int = 140
	// Decode the request body
	type params struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	p := params{}
	err := decoder.Decode(&p)
	if err != nil {
		respondWithError(w, 400, "Error decoding the response")
		return
	}

	if len(p.Body) >= maxChirpSize {
		respondWithError(w, 400, "Chirp is too long")
		return
	}
	// encode the response
	type myResponse struct {
		Valid bool `json:"valid"`
	}
	respondWithJson(w, 200, myResponse{Valid: true})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	log.Printf("%s\n", msg)
	type errResponse struct {
		Error string `json:"error"`
	}
	respondWithJson(w, code, errResponse{
		Error: msg,
	})
}

func respondWithJson(w http.ResponseWriter, code int, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func main() {
	const fileRootPath = "."
	const port = "8080"
	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
	}

	mux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(fileRootPath)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	mux.HandleFunc("GET /api/healthz", http.HandlerFunc(okResponseHandler))
	mux.HandleFunc("GET /admin/metrics", http.HandlerFunc(apiCfg.requestLogger))
	mux.HandleFunc("POST /admin/reset", http.HandlerFunc(apiCfg.reset))
	mux.HandleFunc("POST /api/validate_chirp", http.HandlerFunc(validate))

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Fatal(server.ListenAndServe())
}
