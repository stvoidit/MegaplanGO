package megaplan

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// md5Passord - хэшируем пароль в md5
func md5Passord(p string) string {
	hashPassword := md5.New()
	hashPassword.Write([]byte(p))
	md5String := hex.EncodeToString(hashPassword.Sum(nil))
	return md5String
}

// getOTC - получение временного ключа
func getOTC(domain string, login string, md5password string) (string, error) {
	const uriOTC = "/BumsCommonApiV01/User/createOneTimeKeyAuth.api"
	payload := url.Values{}
	payload.Add("Login", login)
	payload.Add("Password", md5password)
	req, _ := http.NewRequest("POST", domain+uriOTC, strings.NewReader(payload.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	OTCdata := new(struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
		Data struct {
			OneTimeKey string `json:"OneTimeKey"`
		} `json:"data"`
	})
	json.Unmarshal(body, &OTCdata)
	if OTCdata.Data.OneTimeKey == "" {
		errMessage := fmt.Sprintf("Не корректный логин или пароль (%s)", OTCdata.Status.Message)
		myerror := errors.New(errMessage)
		return "", myerror
	}
	return OTCdata.Data.OneTimeKey, nil
}

// getToken - AccessId, SecretKey
func getToken(domain string, login string, md5password string, otc string) (string, string, error) {
	const uriToken = "/BumsCommonApiV01/User/authorize.api"
	payload := url.Values{}
	payload.Add("Login", login)
	payload.Add("Password", md5password)
	payload.Add("OneTimeKey", otc)
	req, _ := http.NewRequest("POST", domain+uriToken, strings.NewReader(payload.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	resp, _ := client.Do(req)
	body, _ := ioutil.ReadAll(resp.Body)
	AccessToken := new(struct {
		Status struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
		Data struct {
			UserID       int    `json:"UserId"`
			EmployeeID   int    `json:"EmployeeId"`
			ContractorID string `json:"ContractorId"`
			AccessID     string `json:"AccessId"`
			SecretKey    string `json:"SecretKey"`
		} `json:"data"`
	})
	json.Unmarshal(body, &AccessToken)
	if AccessToken.Data.AccessID == "" || AccessToken.Data.SecretKey == "" {
		errMessage := fmt.Sprintf("Не корректный логин или пароль, токен доступа не получен (%s)", AccessToken.Status.Message)
		myerror := errors.New(errMessage)
		return "", "", myerror
	}
	return AccessToken.Data.AccessID, AccessToken.Data.SecretKey, nil
}