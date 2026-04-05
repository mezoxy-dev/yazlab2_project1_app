package repository

import (
	"auth-service/models"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserRepository: gerçek MongoDB implementasyonu
// Sadece CRUD işlemlerini içerir, iş mantığı AuthService'de yer alır
type UserRepository struct { // Sadece veritabanında işlem yapar, iş mantığı içermez
	collection *mongo.Collection
}

// NewUserRepository: MongoDB koleksiyonunu alır ve UserRepository döner
func NewUserRepository(collection *mongo.Collection) *UserRepository {
	return &UserRepository{collection: collection}
}

func (r *UserRepository) CreateUser(user models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := r.collection.InsertOne(ctx, user)
	return err
}


func (r *UserRepository) FindByUsername(username string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
 
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}