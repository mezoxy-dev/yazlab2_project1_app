package main
import (
    "fmt"
    "net/http"
)
func main() {
    fmt.Println("Booking Service çalışıyor...")
    http.ListenAndServe(":8082", nil)
}