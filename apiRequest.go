package megaplan

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"
)

const (
	headerContentType    = "Content-Type"
	valueApplicationJSON = "application/json"
)

const (
	typeDateOnly     = "DateOnly"
	typeDateTime     = "DateTime"
	typeDateInterval = "DateInterval"
)

// BuildQueryParams - сборка объекта для запроса
func BuildQueryParams(opts ...QueryBuildingFunc) (qp QueryParams) {
	qp = make(QueryParams)
	for _, opt := range opts {
		opt(qp)
	}
	return qp
}

// QueryBuildingFunc - функция посттроения тела запроса (обычно json для post запроса)
type QueryBuildingFunc func(QueryParams)

// CreateEnity - создать базовую сущность в формате "Мегаплана"
// ! могут быть не описаны крайние или редкоиспользуемые типы
func CreateEnity(contentType string, value any) (qp QueryParams) {
	qp = make(QueryParams, 2)
	qp["contentType"] = contentType
	switch contentType {
	case typeDateOnly:
		t, isTime := value.(time.Time)
		if !isTime {
			return nil
		}
		qp["year"] = t.Year()
		qp["month"] = t.Month() - 1
		qp["day"] = t.Day()
	case typeDateTime:
		t, isTime := value.(time.Time)
		if !isTime {
			return nil
		}
		qp["value"] = t.Format(ISO8601)
	case typeDateInterval:
		// если передается не время, то должно указываться кол-во секунд (актуальная документация мегаплана пишет что миллисекунды - это ошибка)
		switch v := value.(type) {
		case time.Time:
			qp["value"] = v.Second()
		case time.Duration:
			qp["value"] = int(v.Seconds())
		default:
			qp["value"] = v
		}
	default:
		// по умолчанию BaseEntity - это объект с указанием типа и ID
		qp["id"] = value
	}
	return
}

// SetEntityField - добавить поле с сущностью
func SetEntityField(fieldName string, contentType string, value any) (qbf QueryBuildingFunc) {
	return func(qp QueryParams) { qp[fieldName] = CreateEnity(contentType, value) }
}

// SetEntityArray - добавление массива сущностей в поле (например список аудиторов)
func SetEntityArray(field string, ents ...QueryBuildingFunc) QueryBuildingFunc {
	return func(qp QueryParams) {
		if len(ents) == 0 {
			return
		}
		var arr = make([]any, len(ents))
		var tmpParams = make(QueryParams)
		for i, ent := range ents {
			ent(tmpParams)
			arr[i] = tmpParams[field]
		}
		qp[field] = arr
	}
}

// SetRawField - добавить поле с простым типом значения (string, int, etc.)
func SetRawField(field string, value any) QueryBuildingFunc {
	return func(qp QueryParams) { qp[field] = value }
}

// UploadFile - загрузка файла, возвращает обычный http.Response, в ответе стандартная структура ответа + данные для базовой сущности
func (c *ClientV3) UploadFile(filename string, fileReader io.Reader) (*http.Response, error) {
	var buf bytes.Buffer // default 1024 bytes buffer
	var mw = multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("files[]", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, fileReader); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, c.domain, &buf)
	if err != nil {
		return nil, err
	}
	request.URL.Path = "/api/file"
	request.Header.Set(headerContentType, mw.FormDataContentType())
	return c.Do(request)
}

// Do - http.Do + установка обязательных заголовков + декомпрессия ответа, если ответ сжат
func (c *ClientV3) Do(req *http.Request) (*http.Response, error) {
	for h := range c.defaultHeaders.Clone() {
		req.Header.Set(h, c.defaultHeaders.Get(h))
	}
	if _, ok := req.Header[headerContentType]; !ok {
		req.Header.Set(headerContentType, valueApplicationJSON)
	}
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	// slog.Debug("response",
	// 	slog.String("proto", res.Proto),
	// 	slog.Int("status", res.StatusCode),
	// 	slog.String("connection", res.Header.Get("connection")),
	// )
	if err := unzipResponse(res); err != nil {
		return nil, err
	}
	return res, nil
}

func (c ClientV3) makeRequestURL(endpoint string, search QueryParams) string {
	var args string // параметры строки запроса
	if search != nil {
		args = search.QueryEscape()
	}
	return (&url.URL{
		Scheme:   "https",
		Host:     c.domain,
		Path:     endpoint,
		RawQuery: args,
	}).String()
}

// DoRequestAPI - т.к. в v3 параметры запроса для GET (json маршализируется и будет иметь вид: "*?{params}=")
func (c ClientV3) DoRequestAPI(method, endpoint string, search QueryParams, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(
		method,
		c.makeRequestURL(endpoint, search),
		body)
	if err != nil {
		return nil, err
	}
	// slog.Debug("DoRequestAPI.NewRequest",
	// 	slog.String("method", method),
	// 	slog.String("endpoint", request.URL.String()))
	return c.Do(request)
}

// DoRequestAPI - т.к. в v3 параметры запроса для GET (json маршализируется и будет иметь вид: "*?{params}=")
func (c ClientV3) DoCtxRequestAPI(ctx context.Context, method, endpoint string, search QueryParams, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx,
		method,
		c.makeRequestURL(endpoint, search),
		body)
	if err != nil {
		return nil, err
	}
	// slog.Debug("DoCtxRequestAPI.NewRequestWithContext",
	// 	slog.String("method", method),
	// 	slog.String("endpoint", request.URL.String()))
	return c.Do(request)
}
