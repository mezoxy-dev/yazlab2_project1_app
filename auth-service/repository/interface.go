package repository
 
import "auth-service/models"
 
// UserStore: Veri katmanını soyutlayan interface.

// Gerçek uygulama (MongoDB) ve test ortamı (MockRepository) bu interface'i implement eder. 
// Service katmanı sadece UserStore interface'ini görür, somut MongoDB implementasyonunu değil. 
// Bu sayede testlerde gerçek MongoDB'ye ihtiyaç duymadan MockRepository kullanarak hızlıca test yazabiliriz.
type UserStore interface {
	CreateUser(user models.User) error
	FindByUsername(username string) (*models.User, error)
}
 