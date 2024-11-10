// package api

// import (
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	. "k8s/project_highlevel_interface"
// )

// var CompletedRequeests = RequestRetrierChecker{
//     executedRequests: make(map[string]bool),
// }

// func addUserToProjectHandler() {}

// func createProjectHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	var data struct {
// 		ProjectName string `json:"projectName"`
// 		CreatorName string `json:"creatorName"`
// 	}

// 	err := json.NewDecoder(r.Body).Decode(&data)
// 	if err != nil {
// 		http.Error(w, "Bad request", http.StatusBadRequest)
// 		return
// 	}

// 	response :=  CreateProject(data.ProjectName, data.CreatorName)
	
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(response)
// }



// func RequestRerierMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//         log.Printf("Request Method: %s, Request URL: %s", r.Method, r.URL.Path)
// 		requestId := r.Header["request_id"][0]
// 		if (CompletedRequeests.isCompleted(requestId)) {
// 			return
// 		}
		 
//         CompletedRequeests.MarkRequestAsCompleted(requestId)
// 		next.ServeHTTP(w, r)
//     })

// }


// func deleteProjectHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodDelete {
// 		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	projectName := r.URL.Query().Get("projectName")
// 	if projectName == "" {
// 		http.Error(w, "Project name is required", http.StatusBadRequest)
// 		return
// 	}

// 	DeleteProject(projectName)

// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Project deleted successfully"))
// }

// func getProjectConnectInfoHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet {
// 		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	// Parse projectName and username from query parameters
// 	projectName := r.URL.Query().Get("projectName")
// 	username := r.URL.Query().Get("username")
// 	if projectName == "" || username == "" {
// 		http.Error(w, "Project name and username are required", http.StatusBadRequest)
// 		return
// 	}

// 	// getProjectConnectInfo(projectName, username)

// 	w.WriteHeader(http.StatusOK)
// 	w.Write([]byte("Project connection info retrieved"))
// }

// func setUpK8sHelperApi() {
// 	http.Handle("/projects/new", RequestRerierMiddleware(http.HandlerFunc(createProjectHandler)) )
// 	http.HandleFunc("/projects/:id", )
// 	http.HandleFunc("/project/:id/containers/new",)
// 	http.HandleFunc("/projects/:id/containers/container_name") // patch, {payload: new_data} 
// 	http.HandleFunc("/projects/:id/containers/container_name") // delete
// 	http.HandleFunc("/projects/:id/roles/new") // for post {}


// 	http.HandleFunc("/auth")


// 	log.Println("Starting server on :8080...")
// 	log.Fatal(http.ListenAndServe(":8080", nil))
// }










package api


import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/gorilla/mux"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "golang.org/x/crypto/bcrypt"
)

// Custom context key type to avoid collisions
type contextKey string

const (
    userContextKey contextKey = "user"
)

// Models
type User struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    Username  string            `bson:"username" json:"username"`
    Password  string            `bson:"password" json:"-"` // "-" excludes from JSON
    K8sToken  string            `bson:"k8s_token" json:"-"`
    CreatedAt time.Time         `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time         `bson:"updated_at" json:"updated_at"`
}

type Session struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    UserID    primitive.ObjectID `bson:"user_id"`
    Username  string            `bson:"username"`
    Token     string            `bson:"token"`
    ExpiresAt time.Time         `bson:"expires_at"`
    CreatedAt time.Time         `bson:"created_at"`
}

// Request/Response structs
type LoginRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type LoginResponse struct {
    Token string `json:"token"`
}

type CreateUserRequest struct {
    Username string `json:"username"`
    Password string `json:"password"`
    K8sToken string `json:"k8s_token"`
}

// Server struct
type Server struct {
    router      *mux.Router
    db          *mongo.Database
    users       *mongo.Collection
    sessions    *mongo.Collection
}

// Server constructor
func NewServer(db *mongo.Database) *Server {
    s := &Server{
        router:   mux.NewRouter(),
        db:       db,
        users:    db.Collection("users"),
        sessions: db.Collection("sessions"),
    }
    s.setupRoutes()
    return s
}

// Route setup
func (s *Server) setupRoutes() {
    // Public routes (no auth required)
    s.router.HandleFunc("/auth/login", s.handleLogin).Methods("POST")
    s.router.HandleFunc("/auth/register", s.handleRegister).Methods("POST")

    // Protected routes (auth required)
    api := s.router.PathPrefix("/api").Subrouter()
    api.Use(s.authMiddleware)

    api.HandleFunc("/users", s.handleGetUsers).Methods("GET")
    api.HandleFunc("/users/{id}", s.handleGetUser).Methods("GET")
    api.HandleFunc("/users/{id}", s.handleDeleteUser).Methods("DELETE")
    api.HandleFunc("/token", s.handleGetK8sToken).Methods("GET")
}

// Middleware
func (s *Server) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token from Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        // Check bearer token format
        parts := strings.Split(authHeader, " ")
        if len(parts) != 2 || parts[0] != "Bearer" {
            http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
            return
        }
        token := parts[1]

        // Validate session
        session, err := s.validateSession(r.Context(), token)
        if err != nil {
            http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
            return
        }

        // Add user to context
        user, err := s.getUserByID(r.Context(), session.UserID)
        if err != nil {
            http.Error(w, "User not found", http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), userContextKey, user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Session validation
func (s *Server) validateSession(ctx context.Context, token string) (*Session, error) {
    var session Session
    err := s.sessions.FindOne(ctx, bson.M{
        "token": token,
        "expires_at": bson.M{"$gt": time.Now()},
    }).Decode(&session)

    if err != nil {
        return nil, fmt.Errorf("invalid session: %w", err)
    }

    return &session, nil
}

// User retrieval
func (s *Server) getUserByID(ctx context.Context, userID primitive.ObjectID) (*User, error) {
    var user User
    err := s.users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    return &user, nil
}

// Handlers
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Find user
    var user User
    err := s.users.FindOne(r.Context(), bson.M{"username": req.Username}).Decode(&user)
    if err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Check password
    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
        http.Error(w, "Invalid credentials", http.StatusUnauthorized)
        return
    }

    // Create session
    token := primitive.NewObjectID().Hex() // not the best token generator
    session := Session{
        UserID:    user.ID,
        Username:  user.Username,
        Token:     token,
        ExpiresAt: time.Now().Add(24 * time.Hour),
        CreatedAt: time.Now(),
    }

    _, err = s.sessions.InsertOne(r.Context(), session)
    if err != nil {
        http.Error(w, "Error creating session", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Check if user exists
    count, err := s.users.CountDocuments(r.Context(), bson.M{"username": req.Username})
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }
    if count > 0 {
        http.Error(w, "Username already exists", http.StatusConflict)
        return
    }

    // Hash password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Create user
    now := time.Now()
    user := User{
        Username:  req.Username,
        Password:  string(hashedPassword),
        K8sToken:  req.K8sToken,
        CreatedAt: now,
        UpdatedAt: now,
    }

    result, err := s.users.InsertOne(r.Context(), user)
    if err != nil {
        http.Error(w, "Error creating user", http.StatusInternalServerError)
        return
    }

    user.ID = result.InsertedID.(primitive.ObjectID)
    user.Password = "" // Don't send password back
    json.NewEncoder(w).Encode(user)
}

func (s *Server) handleGetUsers(w http.ResponseWriter, r *http.Request) {
    cursor, err := s.users.Find(r.Context(), bson.M{})
    if err != nil {
        http.Error(w, "Error fetching users", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(r.Context())

    var users []User
    if err := cursor.All(r.Context(), &users); err != nil {
        http.Error(w, "Error decoding users", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(users)
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := primitive.ObjectIDFromHex(vars["id"])
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    user, err := s.getUserByID(r.Context(), id)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(user)
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := primitive.ObjectIDFromHex(vars["id"])
    if err != nil {
        http.Error(w, "Invalid user ID", http.StatusBadRequest)
        return
    }

    result, err := s.users.DeleteOne(r.Context(), bson.M{"_id": id})
    if err != nil {
        http.Error(w, "Error deleting user", http.StatusInternalServerError)
        return
    }
    if result.DeletedCount == 0 {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }

    // Delete associated sessions
    _, _ = s.sessions.DeleteMany(r.Context(), bson.M{"user_id": id})

    w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetK8sToken(w http.ResponseWriter, r *http.Request) {
    user := r.Context().Value(userContextKey).(*User)
    json.NewEncoder(w).Encode(map[string]string{"k8s_token": user.K8sToken})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}