package proxy // 单独作为一个服务运行时，要改为 main

import (
	//"fmt"
	"io"
	"log"
	"net/http"
	//"io/ioutil"
)

func main() {
	//// 尝试访问网络
	//url := "http://www.baidu.com"
	//resp, err := http.Get(url)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(string(body))
	//
	//fmt.Println("========================================================")

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
		log.Println("\n")
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
