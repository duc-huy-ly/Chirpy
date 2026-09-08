package main

import (
	"context"
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

	"github.com/duc-huy-ly/Chirpy/internal/auth"
	"github.com/duc-huy-ly/Chirpy/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	datatase       *database.Queries
	platform       string
}

type chirpResponseStruct struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    uuid.UUID `json:"user_id"`
}

type userResponseStruct struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
	if cfg.platform != "dev" {
		respondWithError(w, 403, "Forbidden")
		return
	}
	cfg.fileserverHits = atomic.Int32{}
	err := cfg.datatase.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, 400, "Could not delete all users from database")
		return
	}
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	// Validation
	const maxChirpSize int = 140
	type params struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
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
	listOfNotAllowedWords := []string{"kerfuffle", "sharbert", "fornax"}
	cleanedBody := censorBadWords(listOfNotAllowedWords, decodedParameters.Body)
	newChirpParams := database.CreateChirpParams{
		Body:   cleanedBody,
		UserID: decodedParameters.UserID,
	}
	newChirpInDatabase, err := cfg.datatase.CreateChirp(context.Background(), newChirpParams)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	respondWithJSON(w, 201, chirpResponseStruct{
		ID:        newChirpInDatabase.ID,
		Body:      cleanedBody,
		CreatedAt: newChirpInDatabase.CreatedAt,
		UpdatedAt: newChirpInDatabase.UpdatedAt,
		UserID:    newChirpInDatabase.UserID,
	})
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
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	decodedParameters := params{}
	err := decoder.Decode(&decodedParameters)
	if err != nil {
		respondWithError(w, 400, "Error decoding the email from the request")
		return
	}
	hashedPassword, err := auth.HashPassword(decodedParameters.Password)
	if err != nil {
		respondWithError(w, 400, err.Error())
	}

	newUserParams := database.CreateUserParams{
		Email:          decodedParameters.Email,
		HashedPassword: hashedPassword,
	}
	newUser, err := cfg.datatase.CreateUser(r.Context(), newUserParams)
	if err != nil {
		respondWithError(w, 400, "Error creating new User in database")
		return
	}

	type myResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	respondWithJSON(w, 201, myResponse{
		ID:        uuid.UUID(newUser.ID),
		CreatedAt: newUser.CreatedAt,
		UpdatedAt: newUser.UpdatedAt,
		Email:     newUser.Email,
	})
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type params struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	decodedParams := params{}
	err := decoder.Decode(&decodedParams)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	user, err := cfg.datatase.GetUser(context.Background(), decodedParams.Email)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	validPassword, err := auth.CheckPasswordHash(decodedParams.Password, user.HashedPassword)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	if !validPassword {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	respondWithJSON(w, 200, userResponseStruct{
		user.ID,
		user.CreatedAt,
		user.UpdatedAt,
		user.Email,
	})
}

// Used by the 'GET /api/chirps' endpoint, returns an array of all chirps in ascending created_at order
func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.datatase.GetChirps(context.Background())
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}

	response := make([]chirpResponseStruct, len(chirps))
	for i, chirp := range chirps {
		response[i].ID = chirp.ID
		response[i].Body = chirp.Body
		response[i].UserID = chirp.UserID
		response[i].CreatedAt = chirp.CreatedAt
		response[i].UpdatedAt = chirp.UpdatedAt
	}
	respondWithJSON(w, 200, response)
}

func (cfg *apiConfig) handlerGetChirpFromID(w http.ResponseWriter, r *http.Request) {
	IDString := r.PathValue("chirpID")
	chirpUUID, err := uuid.Parse(IDString)
	if err != nil {
		respondWithError(w, 400, "Error parsing the ID of chirp inside the request")
		return
	}
	chirp, err := cfg.datatase.GetChirpByID(context.Background(), chirpUUID)
	if err != nil {
		respondWithError(w, 404, "Chirp not found")
		return
	}
	respondWithJSON(w, 200, chirpResponseStruct{
		chirp.ID,
		chirp.Body,
		chirp.CreatedAt,
		chirp.UpdatedAt,
		chirp.UserID,
	})
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("error getting the env variables : %s\n", err)
	}
	dbURL := os.Getenv("DB_URL")
	currentPlatform := os.Getenv("PLATFORM")
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
		platform:       currentPlatform,
	}

	mux := http.NewServeMux()
	handler := http.StripPrefix("/app", http.FileServer(http.Dir(fileRootPath)))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(handler))

	mux.HandleFunc("GET /api/healthz", http.HandlerFunc(okResponseHandler))
	mux.HandleFunc("GET /admin/metrics", http.HandlerFunc(apiCfg.requestLogger))
	mux.HandleFunc("POST /admin/reset", http.HandlerFunc(apiCfg.reset))
	mux.HandleFunc("POST /api/chirps", http.HandlerFunc(apiCfg.createChirp))
	mux.HandleFunc("POST /api/users", http.HandlerFunc(apiCfg.createUser))
	mux.HandleFunc("GET /api/chirps", http.HandlerFunc(apiCfg.handlerGetChirps))
	mux.HandleFunc("GET /api/chirps/{chirpID}", http.HandlerFunc(apiCfg.handlerGetChirpFromID))
	mux.HandleFunc("POST /api/login", http.HandlerFunc(apiCfg.handlerLogin))

	server := &http.Server{
		Handler: mux,
		Addr:    ":" + port,
	}
	log.Fatal(server.ListenAndServe())
}
