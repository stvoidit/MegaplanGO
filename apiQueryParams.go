package megaplan

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"time"
)

// ISO8601 - формат даты для api
const ISO8601 = `2006-01-02T15:04:05-07:00`

// QueryParams - параметры запроса
type QueryParams map[string]any

// QueryEscape - urlencode для запроса в строке параметрво
func (qp QueryParams) QueryEscape() string {
	b, _ := qp.ToJSON()
	return url.QueryEscape(string(b))
}

// ToJSON - маршализация параметров в JSON
func (qp QueryParams) ToJSON() ([]byte, error) { return json.Marshal(&qp) }

// ToReader - получить io.Reader для добавление в body часть http.Request
func (qp QueryParams) ToReader() (io.Reader, error) {
	b, err := qp.ToJSON()
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// PrettyPrintJSON - SetIndent для читабельного вывода
func (qp QueryParams) PrettyPrintJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(qp)
}

type DateOnly struct {
	ContentType string `json:"contentType"`
	Day         int    `json:"day"`
	Month       int    `json:"month"`
	Year        int    `json:"year"`
}

func (d DateOnly) ToTime() time.Time {
	return time.Date(d.Year, time.Month(d.Month)+1, d.Day, 0, 0, 0, 0, time.UTC)
}
func NewDateOnly(t time.Time) DateOnly {
	return DateOnly{
		ContentType: typeDateOnly,
		Day:         t.Day(),
		Month:       int(t.Month()) - 1,
		Year:        t.Year(),
	}
}
