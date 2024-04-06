package megaplan

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
)

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
