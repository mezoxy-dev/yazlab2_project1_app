package models

// User: Veritabanındaki kullanıcı kaydını temsil eder
type User struct {
	Username string `json:"username" bson:"username"` // json, HTTP isteklerinde kullanılırken, bson MongoDB ile iletişimde kullanılır.
	Password string `json:"password" bson:"password"`
	Role     string `json:"role" bson:"role"`
}

// RegisterRequest: /register endpoint'ine gelen isteği temsil eder.
type RegisterRequest struct {
	Username	string `json:"username"`     // bson yok çünkü mongodb ye kaydedilmeyecek
	Password	string `json:"password"`    
	AdminSecret	string `json:"admin_secret"` 
}