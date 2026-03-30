package main
import (
    "fmt"
    "net/http"
)
func main() {
    fmt.Println("Event Service çalışıyor...")
    http.ListenAndServe(":8081", nil)
}