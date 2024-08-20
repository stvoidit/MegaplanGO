package megaplan

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DefaultClient - клиент по умаолчанию для API.
var (
	cpus             = runtime.NumCPU()
	DefaultTransport = &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        cpus,
		MaxConnsPerHost:     cpus,
		MaxIdleConnsPerHost: cpus,
	}
	DefaultClient = &http.Client{
		Transport: DefaultTransport,
		Timeout:   time.Minute,
	}
	// DefaultHeaders - заголовок по умолчанию - версия go. Используется при инициализации клиента в NewClient.
	DefaultHeaders = http.Header{"User-Agent": {runtime.Version()}}
)

// NewClient - обертка над http.Client для удобной работы с API v3
func NewClient(domain, token string, opts ...ClientOption) (c *ClientV3) {
	if strings.HasPrefix("http", domain) {
		_URL, _ := url.Parse(domain)
		domain = _URL.Host
	}
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

func (c *ClientV3) Close() { c.client.CloseIdleConnections() }

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
func OptionInsecureSkipVerify(yes bool) ClientOption {
	return func(c *ClientV3) {
		if c.client.Transport != nil {
			if (c.client.Transport.(*http.Transport)).TLSClientConfig == nil {
				(c.client.Transport.(*http.Transport)).TLSClientConfig = &tls.Config{InsecureSkipVerify: yes}
			} else {
				(c.client.Transport.(*http.Transport)).TLSClientConfig.InsecureSkipVerify = yes
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
func OptionEnableAcceptEncodingGzip(yes bool) ClientOption {
	const header = "Accept-Encoding"
	return func(c *ClientV3) {
		if yes {
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

func OptionDisableCookie(yes bool) ClientOption {
	return func(c *ClientV3) {
		if yes {
			c.client.Jar = nil
		} else {
			jar, _ := cookiejar.New(nil)
			c.client.Jar = jar
		}
	}
}

func OptionForceAttemptHTTP2(yes bool) ClientOption {
	return func(c *ClientV3) {
		if c.client.Transport != nil {
			(c.client.Transport.(*http.Transport)).ForceAttemptHTTP2 = yes
		} else {
			c.client.Transport = &http.Transport{ForceAttemptHTTP2: yes}
		}
	}
}

func OptionDisablekeepAlive(yes bool) ClientOption {
	return func(c *ClientV3) {
		if c.client.Transport != nil {
			(c.client.Transport.(*http.Transport)).DisableKeepAlives = yes
		} else {
			c.client.Transport = &http.Transport{DisableKeepAlives: yes}
		}
	}
}
