package megaplan

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

// DefaultClient - клиент по умаолчанию для API.
var (
	cpus          = runtime.NumCPU()
	DefaultClient = &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			MaxIdleConns:        cpus,
			MaxConnsPerHost:     cpus,
			MaxIdleConnsPerHost: cpus,
		},
		Timeout: time.Minute,
	}
	// DefaultHeaders - заголовок по умолчанию - версия go. Используется при инициализации клиента в NewClient.
	DefaultHeaders = http.Header{"User-Agent": {runtime.Version()}}
)

// NewClient - обертка над http.Client для удобной работы с API v3
func NewClient(domain, token string, opts ...ClientOption) (c *ClientV3) {
	c = &ClientV3{
		client:         DefaultClient,
		domain:         domain,
		defaultHeaders: DefaultHeaders,
	}
	c.SetToken(token)
	c.SetOptions(opts...)
	return
}

// ClientV3 - клиент
type ClientV3 struct {
	domain         string
	defaultHeaders http.Header
	client         *http.Client
}

// Do - http.Do + установка обязательных заголовков + декомпрессия ответа, если ответ сжат
func (c *ClientV3) Do(req *http.Request) (*http.Response, error) {
	const ct = "Content-Type"
	for h := range c.defaultHeaders {
		req.Header.Set(h, c.defaultHeaders.Get(h))
	}
	if _, ok := req.Header[ct]; !ok {
		req.Header.Set(ct, "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := unzipResponse(res); err != nil {
		return nil, err
	}
	return res, nil
}

// DoRequestAPI - т.к. в v3 параметры запроса для GET (json маршализируется и будет иметь вид: "*?{params}=")
func (c ClientV3) DoRequestAPI(method string, endpoint string, search QueryParams, body io.Reader) (*http.Response, error) {
	var args string // параметры строки запроса
	if search != nil {
		args = search.QueryEscape()
	}
	request, err := http.NewRequest(method, c.domain, body)
	if err != nil {
		return nil, err
	}
	request.URL.Path = endpoint
	request.URL.RawQuery = args
	return c.Do(request)
}

// DoRequestAPI - т.к. в v3 параметры запроса для GET (json маршализируется и будет иметь вид: "*?{params}=")
func (c ClientV3) DoCtxRequestAPI(ctx context.Context, method string, endpoint string, search QueryParams, body io.Reader) (*http.Response, error) {
	var args string // параметры строки запроса
	if search != nil {
		args = search.QueryEscape()
	}
	request, err := http.NewRequestWithContext(ctx, method, c.domain, body)
	if err != nil {
		return nil, err
	}
	request.URL.Path = endpoint
	request.URL.RawQuery = args
	return c.Do(request)
}

// ErrUnknownCompressionMethod - неизвестное значение в заголовке "Content-Encoding"
// не является фатальной ошибкой, должна возвращаться вместе с http.Response.Body,
// чтобы пользователь мог реализовать свой метод обработки сжатого сообщения
var ErrUnknownCompressionMethod = errors.New("unknown compression method")

// unzipResponse - распаковка сжатого ответа
func unzipResponse(response *http.Response) (err error) {
	if response.Uncompressed {
		return nil
	}
	switch response.Header.Get("Content-Encoding") {
	case "":
		return nil
	case "gzip":
		gz, err := gzip.NewReader(response.Body)
		if err != nil {
			return err
		}
		b, err := io.ReadAll(gz)
		if err != nil {
			return err
		}
		if err := response.Body.Close(); err != nil {
			return err
		}
		if err := gz.Close(); err != nil {
			return err
		}
		response.Body = io.NopCloser(bytes.NewReader(b))
		response.Header.Del("Content-Encoding")
		response.Uncompressed = true
		return nil
	default:
		return ErrUnknownCompressionMethod
	}
}

// SetOptions - применить опции
func (c *ClientV3) SetOptions(opts ...ClientOption) {
	for i := range opts {
		opts[i](c)
	}
}

// SetToken - установить или изменить токен доступа
func (c *ClientV3) SetToken(token string) { c.defaultHeaders.Set("AUTHORIZATION", "Bearer "+token) }

// ClientOption - функция применения настроект
type ClientOption func(*ClientV3)

// OptionInsecureSkipVerify - переключение флага bool в http.Client.Transport.TLSClientConfig.InsecureSkipVerify - отключать или нет проверку сертификтов
// Если домен использует самоподписанные сертифика, то удобно включать на время отладки и разработки
func OptionInsecureSkipVerify(b bool) ClientOption {
	return func(c *ClientV3) {
		if c.client.Transport != nil {
			if (c.client.Transport.(*http.Transport)).TLSClientConfig == nil {
				(c.client.Transport.(*http.Transport)).TLSClientConfig = &tls.Config{InsecureSkipVerify: b}
			} else {
				(c.client.Transport.(*http.Transport)).TLSClientConfig.InsecureSkipVerify = b
			}
		}
	}
}

// OptionsSetHTTPTransport - установить своб настройку http.Transport
func OptionsSetHTTPTransport(tr http.RoundTripper) ClientOption {
	return func(c *ClientV3) { c.client.Transport = tr }
}

// OptionEnableAcceptEncodingGzip - доабвить заголов Accept-Encoding=gzip к запросу
// т.е. объекм трафика на хуках может быть большим, то удобно запрашивать сжатый ответ
func OptionEnableAcceptEncodingGzip(b bool) ClientOption {
	const header = "Accept-Encoding"
	return func(c *ClientV3) {
		if b {
			c.defaultHeaders.Set(header, "gzip")
		} else {
			c.defaultHeaders.Del(header)
		}
	}
}

// OptionSetClientHTTP - установить свой экземпляр httpClient
func OptionSetClientHTTP(client *http.Client) ClientOption {
	return func(c *ClientV3) {
		if client != nil {
			c.client = client
		}
	}
}

// OptionSetXUserID - добавить заголовок "X-User-Id" - запросы будут выполнятся от имени указанного пользователя.
// Если передано значение <= 0, то заголовок будет удален
func OptionSetXUserID(userID int) ClientOption {
	const header = "X-User-Id"
	return func(c *ClientV3) {
		if userID > 0 {
			c.defaultHeaders.Set(header, strconv.Itoa(userID))
		} else {
			c.defaultHeaders.Del(header)
		}
	}
}
