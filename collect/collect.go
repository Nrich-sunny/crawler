package collect

import (
	"bufio"
	"fmt"
	"github.com/Nrich-sunny/crawler/extensions"
	"github.com/Nrich-sunny/crawler/proxy"
	"go.uber.org/zap"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
	"io"
	"net/http"
	"time"
)

type Fetcher interface {
	Get(req *Request) ([]byte, error)
}

type BaseFetch struct {
}

func (BaseFetch) Get(req *Request) ([]byte, error) {

	resp, err := http.Get(req.Url)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error status code: %v\n", resp.StatusCode)
	}

	bodyReader := bufio.NewReader(resp.Body)
	e := DetermineEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return io.ReadAll(utf8Reader)
}

// 模拟浏览器访问
type BrowserFetch struct {
	Timeout time.Duration
	Proxy   proxy.ProxyFunc // 是 Transport 结构体中的函数
	Logger  *zap.Logger
}

func (b BrowserFetch) Get(request *Request) ([]byte, error) {
	client := &http.Client{
		Timeout: b.Timeout,
	}

	if b.Proxy != nil {
		transport := http.DefaultTransport.(*http.Transport)
		transport.Proxy = b.Proxy // 将其替换为自定义的代理函数
		client.Transport = transport
	}

	req, err := http.NewRequest("GET", request.Url, nil)
	if err != nil {
		return nil, fmt.Errorf("get url failed:%v", err)
	}

	if len(request.Task.Cookie) > 0 {
		req.Header.Set("Cookie", request.Task.Cookie)
	}
	//req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/105.0.0.0 Safari/537.36")
	req.Header.Set("User-Agent", extensions.GenerateRandomUA())

	resp, err := client.Do(req)
	time.Sleep(request.Task.WaitTime)
	if err != nil {
		b.Logger.Error("fetch failed", zap.Error(err))
		return nil, err
	}

	bodyReader := bufio.NewReader(resp.Body)
	e := DetermineEncoding(bodyReader)
	utf8Reader := transform.NewReader(bodyReader, e.NewDecoder())
	return io.ReadAll(utf8Reader)
}

func DetermineEncoding(r *bufio.Reader) encoding.Encoding {

	bytes, err := r.Peek(1024)

	if err != nil {
		fmt.Printf("fetch error:%v\n", err)
		return unicode.UTF8
	}

	e, _, _ := charset.DetermineEncoding(bytes, "")
	return e
}
