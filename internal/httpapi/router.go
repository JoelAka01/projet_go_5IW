package httpapi

import "net/http"

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth", handleAuth)
	mux.HandleFunc("/products", handleProducts)
	mux.HandleFunc("/cart", handleCart)
	mux.HandleFunc("/orders", handleOrders)
	return mux
}
