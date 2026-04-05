package main
 
import (
	"context"
	"dispatcher/middleware"
	"dispatcher/proxy"
	"dispatcher/repository"
	"dispatcher/service"
	"log"
	"net/http"
	"os"
	"time"
 
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")

	client := connectMongo(mongoURI)
	trafficColl := client.Database("dispatcher_logs").Collection("traffic")

	// Logları tarihe göre tersten çekiyoruz (Admin UI), bu yüzden index şart
	_, _ = trafficColl.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{{Key: "timestamp", Value: -1}},
	})

	logRepo  := repository.NewMongoLogRepository(trafficColl)
	logSvc   := service.NewLogService(logRepo)
	authSvc  := service.NewAuthService()
	proxySvc := service.NewProxyService(authSvc)

	router := proxy.SetupRouter(proxySvc, logSvc)

	handler := middleware.CORS(
		middleware.Logger(logSvc)(
			middleware.Auth(authSvc)(router),
		),
	)

	log.Println("Dispatcher :8080 portunda başlatılıyor...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Sunucu hatası: %v", err)
	}
}

func connectMongo(uri string) *mongo.Client {
	var client *mongo.Client
	var err error
 
	for i := 0; i < 5; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err == nil {
			err = client.Ping(ctx, nil)
		}
		cancel()
 
		if err == nil {
			return client
		}
		log.Printf("MongoDB henüz hazır değil, bekleniyor... (%d/5)", i+1)
		time.Sleep(2 * time.Second)
	}
 
	log.Fatalf("MongoDB bağlantı hatası: %v", err)
	return nil
}
 