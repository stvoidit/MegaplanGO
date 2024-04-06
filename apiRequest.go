package megaplan

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

// ISO8601 - формат даты для api
const ISO8601 = `2006-01-02T15:04:05-07:00`

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
	case "DateOnly":
		t, isTime := value.(time.Time)
		if !isTime {
			return nil
		}
		qp["year"] = t.Year()
		qp["month"] = t.Month() - 1
		qp["day"] = t.Day()
	case "DateTime":
		t, isTime := value.(time.Time)
		if !isTime {
			return nil
		}
		qp["value"] = t.Format(ISO8601)
	case "DateInterval":
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
	request.Header.Set("Content-Type", mw.FormDataContentType())
	return c.Do(request)
}
