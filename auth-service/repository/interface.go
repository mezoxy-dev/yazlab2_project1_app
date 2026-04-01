package repository
 
import "auth-service/models"
 
// UserStore: Veri katmanını soyutlayan interface.
//
// Gerçek uygulama (MongoDB) ve test ortamı (MockRepository) bu
// interface'i implement eder. Sayesinde service katmanı hiçbir zaman
// doğrudan veritabanına bağımlı olmaz — sadece bu sözleşmeye bağımlıdır.
type UserStore interface {
	CreateUser(user models.User) error
	FindByUsername(username string) (*models.User, error)
}
 