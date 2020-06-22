package megaplan

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const rfc2822 = "Mon, 02 Jan 2006 15:04:05 -0700"

// GET - get запрос к API
func (api *API) GET(uri string, payload map[string]interface{}) *ResponseBuffer {
	const rMethod = "GET"
	urlQuery, urlParams, queryHeader := api.queryHasher(rMethod, uri, payload)
	return api.requestQuery(rMethod, urlQuery, urlParams, queryHeader)
}

// POST - post запрос на API
func (api *API) POST(uri string, payload map[string]interface{}) *ResponseBuffer {
	const rMethod = "POST"
	urlQuery, urlParams, queryHeader := api.queryHasher(rMethod, uri, payload)
	return api.requestQuery(rMethod, urlQuery, urlParams, queryHeader)
}

// CheckUser - проверка пользователя для встроенного приложения
func (api *API) CheckUser(userSign string) (UserAppVerification, error) {
	var appAPI API
	appAPI = *api
	appAPI.accessID = api.appUUID
	appAPI.secretKey = api.appSecret
	var payload = map[string]interface{}{
		"uuid":     api.appUUID,
		"userSign": userSign,
	}
	b := appAPI.POST("/BumsSettingsApiV01/Application/checkUserSign.json", payload)
	var i UserVerifyResponse
	if err := json.NewDecoder(b).Decode(&i); err != nil {
		return i.Data, err
	}
	if i.response.Status.Code != "ok" {
		return i.Data, errors.New("invalide user data")
	}
	return i.Data, nil
}

// queryHasher - задаем сигнатуру, отдает URL и Header для запросов к API
func (api *API) queryHasher(method string, uri string, payload map[string]interface{}) (*url.URL, url.Values, http.Header) {
	var urlParams = make(url.Values)
	for k, val := range payload {
		switch t := val.(type) {
		case uint, uint32, uint64, int, int32, int64:
			urlParams.Add(k, fmt.Sprintf("%d", t))
		case bool:
			urlParams.Add(k, strconv.FormatBool(t))
		case string:
			urlParams.Add(k, t)
		case nil:
			continue
		default:
			fmt.Println("unrecognized type", t)
		}
	}
	URL, err := url.Parse(api.domain)
	if err != nil {
		panic(err.Error())
	}
	URL.Path += uri
	today := time.Now().Format(rfc2822)
	if len(urlParams) > 0 && method == "GET" {
		URL.RawQuery = urlParams.Encode()
	}
	sigURL := strings.Replace(URL.String(), fmt.Sprintf("%s://", URL.Scheme), "", 1)
	Signature := fmt.Sprintf("%s\n\napplication/x-www-form-urlencoded\n%s\n%s", method, today, sigURL)
	h := hmac.New(sha1.New, api.secretKey)
	h.Write([]byte(Signature))
	hexSha1 := hex.EncodeToString(h.Sum(nil))
	sha1Query := base64.StdEncoding.EncodeToString([]byte(hexSha1))
	queryHeader := http.Header{
		"Date":            []string{today},
		"X-Authorization": []string{api.accessID + ":" + sha1Query},
		"Accept":          []string{"application/json"},
		"Content-Type":    []string{"application/x-www-form-urlencoded"},
		"accept-encoding": []string{"gzip, deflate, br"},
	}
	return URL, urlParams, queryHeader
}

// requestQuery - итоговый запрос к API предварительно сформированный Request с правильным набором headers
func (api *API) requestQuery(method string, URL *url.URL, urlParams url.Values, headers http.Header) *ResponseBuffer {
	var req *http.Request
	var err error
	if method == "POST" {
		if req, err = http.NewRequest(method, URL.String(), strings.NewReader(urlParams.Encode())); err != nil {
			panic(err.Error())
		}
	} else {
		if req, err = http.NewRequest(method, URL.String(), nil); err != nil {
			panic(err.Error())
		}
		req.URL.RawQuery = urlParams.Encode()
	}
	req.Header = headers
	resp, err := api.client.Do(req)
	if err != nil {
		panic(err.Error())
	}
	var buff = new(ResponseBuffer)
	if _, err := buff.ReadFrom(resp.Body); err != nil {
		panic(err.Error())
	}
	resp.Body.Close()
	return buff
}