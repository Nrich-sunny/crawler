package proxy // 单独作为一个服务运行时，要改为 main

import (
	"io"
	"log"
	"net/http"
)

func main() {
	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleHTTP(w, r)
		}),
	}
	log.Println("server starting...")
	log.Fatal(server.ListenAndServe())
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL)
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.Println("error in proxy-server...")
		log.Printf("status service unavailable code: %v\n", http.StatusServiceUnavailable)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
