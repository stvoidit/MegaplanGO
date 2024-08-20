package megaplan

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Response - ответ API
type Response[T any] struct {
	Meta Meta `json:"meta"` // metainfo ответа
	Data T    `json:"data"` // поле для декодирования присвоенной структуры
}

// Next - есть ли следующая страница
func (res Response[T]) Next() bool { return res.Meta.Pagination.HasMoreNext }

// Prev - есть ли предыдущая страница
func (res Response[T]) Prev() bool { return res.Meta.Pagination.HasMorePrev }

// Decode - парсинг ответа API
func (res *Response[T]) Decode(r *http.Response) (err error) {
	defer r.Body.Close()
	if !strings.Contains(r.Header.Get(headerContentType), valueApplicationJSON) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		return errors.New(string(b))
	}
	if err := json.NewDecoder(r.Body).Decode(res); err != nil {
		return err
	}
	return res.Meta.Error()
}

// Pagination - пагинация
type Pagination struct {
	Count       int64 `json:"count"`
	Limit       int64 `json:"limit"`
	CurrentPage int64 `json:"currentPage"`
	HasMoreNext bool  `json:"hasMoreNext"`
	HasMorePrev bool  `json:"hasMorePrev"`
}

// UnmarshalJSON - json.Unmarshaler
func (p *Pagination) UnmarshalJSON(b []byte) (err error) {
	if bytes.Equal(b, []byte{91, 93}) {
		return nil
	}

	dec := json.NewDecoder(bytes.NewReader(b))
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if _, isdelim := t.(json.Delim); isdelim {
			continue
		}
		if field, ok := t.(string); ok {
			switch field {
			case "count":
				err = dec.Decode(&p.Count)
			case "limit":
				err = dec.Decode(&p.Limit)
			case "currentPage":
				err = dec.Decode(&p.CurrentPage)
			case "hasMoreNext":
				err = dec.Decode(&p.HasMoreNext)
			case "hasMorePrev":
				err = dec.Decode(&p.HasMorePrev)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// MarshalJSON - json.Marshaler
// TODO: вообще обратный маршалинг на практике не нужен, поэтому нужно доделать позже
func (p Pagination) MarshalJSON() ([]byte, error) { return nil, nil }

// Meta - metainfo
type Meta struct {
	Errors []struct {
		Fields  any `json:"field"`
		Message any `json:"message"`
	} `json:"errors"`
	Status     int        `json:"status"`
	Pagination Pagination `json:"pagination"`
}

// Error - если была ошибка, переданная в meta, то вернется ошибка с описание мегаплана, если нет, то вернется nil
func (m Meta) Error() (err error) {
	if len(m.Errors) > 0 {
		var errorsStr = make([]string, len(m.Errors))
		for i := range m.Errors {
			errorsStr[i] = fmt.Sprintf("FIELD: %v MESSAGE: %v", m.Errors[i].Fields, m.Errors[i].Message)
		}
		err = errors.New(strings.Join(errorsStr, "\n"))
	}
	return
}

// ParseResponse - utility-функция для упрощения чтения ответа API
func ParseResponse[T any](r *http.Response) (res Response[T], err error) {
	defer r.Body.Close()
	if !strings.Contains(r.Header.Get(headerContentType), valueApplicationJSON) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return res, err
		}
		return res, errors.New(string(b))
	}
	e := json.NewDecoder(r.Body)
	e.UseNumber()
	err = e.Decode(&res)
	return
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
		defer response.Body.Close()
		b, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		gz, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return err
		}
		response.Body = gz
		response.Header.Del("Content-Encoding")
		response.Uncompressed = true
		return nil
	default:
		return ErrUnknownCompressionMethod
	}
}
