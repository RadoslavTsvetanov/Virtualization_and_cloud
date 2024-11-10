package api 

import (
    "context"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "log"
    "math/rand"
    "net/http"
    "os"
    "time"
"github.com/gin-contrib/cors"
    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var usersCollection *mongo.Collection

type User struct {
    Username  string `json:"username" bson:"username"`
    Password  string `json:"password" bson:"password"`
    K8sToken  string `json:"k8sToken" bson:"k8sToken"`
}

type SessionManager struct {
    sessions map[string]string
}

var sessionManager = &SessionManager{
    sessions: make(map[string]string),
}

func connectToDB() {
    mongoURI := os.Getenv("MONGO_URI") // Get MongoDB URI from environment variable
    if mongoURI == "" {
        log.Fatal("MONGO_URI environment variable is not set")
    }

    clientOptions := options.Client().ApplyURI(mongoURI)
    var err error
    client, err = mongo.Connect(context.Background(), clientOptions)
    if err != nil {
        log.Fatal(err)
    }

    usersCollection = client.Database("test").Collection("users")
}

func hashPassword(password string) string {
    hash := sha256.New()
    hash.Write([]byte(password))
    return base64.URLEncoding.EncodeToString(hash.Sum(nil))
}

func generateSessionToken() string {
    rand.Seed(time.Now().UnixNano())
    token := fmt.Sprintf("%d", rand.Int63())
    return base64.URLEncoding.EncodeToString([]byte(token))
}

func checkIfTokenBelongsToUser(username, token string) bool {
    if storedToken, exists := sessionManager.sessions[username]; exists {
        return storedToken == token
    }
    return false
}

func issueTokenForUser(username string) string {
    token := generateSessionToken()
    sessionManager.sessions[username] = token
    return token
}

func signup(c *gin.Context) {
    var user User
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
        return
    }

    var existingUser User
    err := usersCollection.FindOne(context.Background(), bson.M{"username": user.Username}).Decode(&existingUser)
    if err == nil {
        c.JSON(http.StatusConflict, gin.H{"message": "User already exists"})
        return
    }

    user.Password = hashPassword(user.Password)
    _, err = usersCollection.InsertOne(context.Background(), user)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "User created successfully"})
}

func login(c *gin.Context) {
    var request struct {
        Username string `json:"username"`
        Password string `json:"password"`
    }
    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
        return
    }

    var user User
    err := usersCollection.FindOne(context.Background(), bson.M{"username": request.Username}).Decode(&user)
    if err != nil || user.Password != hashPassword(request.Password) {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid username or password"})
        return
    }

    token := issueTokenForUser(user.Username)
    c.JSON(http.StatusOK, gin.H{"token": token})
}

func getK8sToken(c *gin.Context) {
    username := c.Param("username")
    token := c.GetHeader("Authorization")

    if !checkIfTokenBelongsToUser(username, token) {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
        return
    }

    var user User
    err := usersCollection.FindOne(context.Background(), bson.M{"username": username}).Decode(&user)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"k8sToken": user.K8sToken})
}

func setK8sToken(c *gin.Context) {
    username := c.Param("username")
    token := c.GetHeader("Authorization")

    if !checkIfTokenBelongsToUser(username, token) {
        c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid token"})
        return
    }

    var request struct {
        K8sToken string `json:"k8sToken"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
        return
    }

    _, err := usersCollection.UpdateOne(
        context.Background(),
        bson.M{"username": username},
        bson.M{"$set": bson.M{"k8sToken": request.K8sToken}},
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating K8s token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "K8s token updated"})
}

func SetupServer() {
    if err := godotenv.Load(); err != nil {
        log.Fatal("Error loading .env file")
    }

    connectToDB()
    defer client.Disconnect(context.Background())

    r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},   // Allow all origins
		AllowMethods:     []string{"*"},   // Allow all HTTP methods
		AllowHeaders:     []string{"*"},   // Allow all headers
		ExposeHeaders:    []string{"*"},   // Expose all headers
		AllowCredentials: true,            // Allow credentials (cookies, etc.)
	}))
    r.POST("/signup", signup)
    r.POST("/login", login)
    r.GET("/getK8sToken/:username", getK8sToken)
    r.POST("/setK8sToken/:username", setK8sToken)

    r.Run(":8080")
}

