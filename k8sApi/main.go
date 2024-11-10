package main

import (
	"fmt"
	// . "k8s/primitives"
	. "k8s/templates"
    "context"
    "log"
    "net/http"
    "os"
    "time"
 "go.mongodb.org/mongo-driver/bson"
    "github.com/joho/godotenv"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
	. "k8s/api"
)


var namespace = "alerts-workloads-manager-system"

func testApplyingTemplateContainerWithNormalValues() {

	tc := TemplateContainer{
		ImageName: "nginx",
		Name:      "nginx-container",
		Exposure:  Exposure{Type: Exposed},
		Env: map[string]string{
			"FOO": "bar",
		},
		Mode: Managed,
		Metadata: TemplateMetadata{
			Author:      "John Doe",
			Version:     "1.0.0",
			Description: "Nginx web server container",
		},
		Labels: map[string]string{
			"app": "nginx",
		},
		Port: 80,
	}

	err := ApplyTemplateToProject(namespace, []TemplateContainer{tc})
	if err != nil {
		fmt.Println(err)
	}

}

func testingSpinningUpMiniInfra() {

	tc1 := TemplateContainer{
		ImageName: "postgres",
		Name:      "my-postgres",
		Exposure:  Exposure{Type: Internal},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "kl4fr9fUDS",
			"POSTGRES_DB":       "postgres",
			"POSTGRES_HOST":     "my-release-postgresql",
			"POSTGRES_PORT":     "5432",
		},
		Mode: Unmanaged,
		Metadata: TemplateMetadata{
			Author:      "John Doe",
			Version:     "1.0.0",
			Description: "PostgreSQL database container",
		},
		Labels: map[string]string{
			"app": "back-up-db",
		},
		Port: 5432,
	}

	tc2 := TemplateContainer{
		ImageName: "radoslav123/temp-temp:latest",
		Name:      "my-server",
		Exposure:  Exposure{Type: Exposed},
		Env: map[string]string{
			"PG_HOST":     "primary-db-v-2",
			"PG_PORT":     "5432",
			"PG_USER":     "postgres",
			"PG_PASSWORD": "kl4fr9fUDS",
			"PG_DATABASE": "postgres",
		},
		Mode: Managed,
		Metadata: TemplateMetadata{
			Author:      "John Doe",
			Version:     "1.0.0",
			Description: "Node.js server container",
		},
		Labels: map[string]string{
			"app": "server",
		},
		Port: 3000,
	}

	ApplyTemplateToProject("ooo", []TemplateContainer{tc1, tc2})

}


func setUpServerAndEnvs(){
if err := godotenv.Load(); err != nil {
        log.Printf("Warning: .env file not found")
    }

    // Get MongoDB connection string from environment variable
    mongoURI := getEnvWithDefault("MONGODB_URI", "mongodb://localhost:27017")
    dbName := getEnvWithDefault("DB_NAME", "userauth")
    port := getEnvWithDefault("PORT", "8080")

    // Connect to MongoDB
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)

    // Ping the database
    if err := client.Ping(ctx, nil); err != nil {
        log.Fatal(err)
    }

    // Create indexes
    db := client.Database(dbName)
    createIndexes(ctx, db)

    // Initialize and start server
    server := NewServer(db)
    
    log.Printf("Server starting on port %s", port)
    if err := http.ListenAndServe(":"+port, server); err != nil {
        log.Fatal(err)
    }
}

func getEnvWithDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func createIndexes(ctx context.Context, db *mongo.Database) {
    // Create unique index on username
    _, err := db.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "username", Value: 1}},
        Options: options.Index().SetUnique(true),
    })
    if err != nil {
        log.Fatal(err)
    }

    // Create index on session token and expiry
    _, err = db.Collection("sessions").Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{
            {Key: "token", Value: 1},
            {Key: "expires_at", Value: 1},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}

func main() {

	// namespace := "testing-rbac-custom-roles"

	// fmt.Print(GetServicesFromOutsideTheCluster(namespace))

	// testApplyingTemplateContainerWithNormalValues()

	// envVars := map[string]string{
	// 	"POSTGRES_USER":     "postgres",
	// 	"POSTGRES_PASSWORD": "kl4fr9fUDS",
	// 	"POSTGRES_DB":       "postgres",
	// 	"POSTGRES_HOST":     "my-release-postgresql",
	// 	"POSTGRES_PORT":     "5432",
	// }
	// labels := map[string]string{
	// 	"app": "example-app",
	// 	"env": "development",
	// }

	// err := CreateNamespace(namespace)
	// DefaultHandleError(err)

	// queryAllResources(namespace)
	// DefaultHandleError(CreateNamespaceRestrictedUser(namespace, "normal-user"))
	// CreateNamespace(namespace)
	// e := CreateRole("admin", namespace, []rbacv1.PolicyRule{
	// 	{
	// 		APIGroups: []string{"", "extensions", "apps"},
	// 		Resources: []string{"*"},
	// 		Verbs:     []string{"*"},
	// 	},
	// 	{
	// 		APIGroups: []string{"batch"},
	// 		Resources: []string{"jobs", "cronjobs"},
	// 		Verbs:     []string{"*"},
	// 	},
	// })

	// DefaultHandleError(e)
	// e = CreateServiceAccount("pesho", "admin", namespace)

	// token, err := GetUserToken(namespace, "pesho")
	// fmt.Println(token)
	// DefaultHandleError(err)

	// fmt.Println(createProject("huihuiov", "huiyo"))

	// wwwww()

	// fffff()

	// exposeService()

	// gettingEndpointsInAProjectForOutsideProjectAccess()

	// just_for_testing_workload_operator()

// 	fmt.Print("")

// 	GetServicesFromOutsideTheCluster("ooo")
// testApplyingTemplateContainerWithNormalValues()
	// testingSpinningUpMiniInfra()


	setUpServerAndEnvs()

}
