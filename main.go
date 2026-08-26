package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"
	"uuid"

	"github.com/duc-huy-ly/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	datatase       *database.Queries
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
	decodedParameters := params{}
	err := decoder.Decode(&decodedParameters)
	if err != nil {
		respondWithError(w, 400, "Error decoding the response")
		return
	}

	if len(decodedParameters.Body) >= maxChirpSize {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	// Replace all profane words with static 4 char string ****
	listOfNotAllowedWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleanedBody := censorBadWords(listOfNotAllowedWords, decodedParameters.Body)
	// encode the response
	type myResponse struct {
		Body string `json:"cleaned_body"`
	}
	respondWithJSON(w, 200, myResponse{Body: cleanedBody})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	log.Printf("%s\n", msg)
	type errResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
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

func censorBadWords(notAllowedWords []string, body string) string {
	splitBody := strings.Split(body, " ")
	result := make([]string, 0)
	for _, word := range splitBody {
		wordToLower := strings.ToLower(word)
		if slices.Contains(notAllowedWords, wordToLower) {
			result = append(result, "****")
			continue
		}
		result = append(result, word)
	}
	return strings.Join(result, " ")
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	// accepts an email in the request body
	type params struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	decodedParameters := params{}
	err := decoder.Decode(&decodedParameters)
	if err != nil {
		respondWithError(w, 400, "Error decoding the email from the request")
		return
	}
	newUser, err := cfg.datatase.CreateUser(r.Context(), decodedParameters.Email)
	if err != nil {
		respondWithError(w, 400, "Error creating new User in database")
		return
	}

	// Response time

	type myRespnse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	respondWithJSON(w, 200, myRespnse{
		ID:        uuid.UUID(newUser.ID),
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error getting the env variables : %s\n", err)
	}
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("error opening databse : %s\n", err)
		return
	}
	dbQueries := database.New(db)

	const fileRootPath = "."
	const port = "8080"

	apiCfg := &apiConfig{
		fileserverHits: atomic.Int32{},
		datatase:       dbQueries,
	}

	mux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(fileRootPath)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	mux.HandleFunc("GET /api/healthz", http.HandlerFunc(okResponseHandler))
	mux.HandleFunc("GET /admin/metrics", http.HandlerFunc(apiCfg.requestLogger))
	mux.HandleFunc("POST /admin/reset", http.HandlerFunc(apiCfg.reset))
	mux.HandleFunc("POST /api/validate_chirp", http.HandlerFunc(validate))
	mux.HandleFunc("POST /api/createUser", http.HandlerFunc(apiCfg.createUser))

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Fatal(server.ListenAndServe())
}
