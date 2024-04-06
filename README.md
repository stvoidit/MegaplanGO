# Пример использования

Иниализация клиента + опция включения заголовка "Accept-Encoding":"gzip", ответ будет возвращаться сжатым, реализована декомпрессия тела ответа внутри вызова.

```golang
    import (
        "github.com/stvoidit/megaplan/v3"
    )
    const (
        domain = `https://yourdomain.ru`
        token  = `token`
    )
    func main() {
        client := megaplan.NewClient(domain, token, megaplan.OptionEnableAcceptEncodingGzip(true))
    }
```
## Пример создания задачами
https://demo.megaplan.ru/api/v3/docs#entityTask
Для удобства составления json для тела запроса есть функция __megaplan.BuildQueryParams__. Её единственное назначение - собрать параметры в правильном формате.
Некоторые сущности требуют специального формата (например [Дата и Время](https://demo.megaplan.ru/api/v3/docs#entityDateTime), [Интервал](https://demo.megaplan.ru/api/v3/docs#entityDateInterval), [Дата](https://demo.megaplan.ru/api/v3/docs#entityDateOnly), [~~Сдвиг дат~~](https://demo.megaplan.ru/api/v3/docs#entityShiftDate)), то функция __megaplan.BuildQueryParams__ корректно сформирует структуру этих сущностей.


```golang
func CreateTask(c *megaplan.ClientV3) {
    const endpoint = "/api/v3/task"
    var qp = megaplan.BuildQueryParams(
        megaplan.SetRawField("contentType", "Task"),
        megaplan.SetRawField("isUrgent", false),
        megaplan.SetRawField("isTemplate", false),
        megaplan.SetRawField("name", "library test"),
        megaplan.SetRawField("subject", "subject library test"),
        megaplan.SetRawField("statement", "statement library test"),
        megaplan.SetEntityField("owner", "Employee", 1000129),
        megaplan.SetEntityField("responsible", "Employee", 1000129),
        megaplan.SetEntityField("deadline", "DateOnly", time.Now().Add(time.Hour*72)),
        megaplan.SetEntityField("plannedWork", "DateInterval", time.Hour*13),
    )
    r, err := qp.ToReader()
    if err != nil {
        panic(err)
    }
    rc, err := c.DoRequestAPI(http.MethodPost, endpoint, nil, r)
    if err != nil {
        panic(err)
    }
    defer rc.Close()
    os.Stdout.ReadFrom(rc)
}
```
## Пример запроса с параметрами URL
Так как параметры запроса на api "Мегаплан" передаются в нетипичном формате ("*?json=?"), то необходимо их экранировать через url.QueryEscape.
Для удобства составления этих параметров можно так же использовать тип __megaplan.QueryParams__.
```golang
    func testGetWithFilters(c *megaplan.ClientV3) {
        const endpoint = "/api/v3/task"
        var requestedFiled = [...]string{
            "id",
            "name",
            "status",
            "deadline",
            "actualWork",
            "responsible",
            "timeCreated",
        }
        // параметры верхнего уровня
        var searchParams = megaplan.BuildQueryParams(
            megaplan.SetRawField("limit", 50),
            megaplan.SetRawField("onlyRequestedFields", true),
            megaplan.SetRawField("fields", requestedFiled),
        )

        // пример составления параметров без megaplan.BuildQueryParams (т.к. есть большая вложенность параметров)
        // megaplan.QueryParams - это просто алиас к типа megaplan.QueryParams, но с доп. методами,
        // поэтому для корректного составления json в параметрах URL необходимо передавать в DoRequestAPI именно megaplan.QueryParams
        now := time.Now()
        from := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.Local)
        var filterParams = map[string]interface{}{
            "contentType": "TaskFilter",
            "id":          nil,
            "config": megaplan.QueryParams{
                "contentType": "FilterConfig",
                "termGroup": megaplan.QueryParams{
                    "contentType": "FilterTermGroup",
                    "join":        "and",
                    "terms": [...]megaplan.QueryParams{
                        {
                            "contentType": "FilterTermEnum",
                            "field":       "status",
                            "comparison":  "equals",
                            "value":       [...]string{"filter_any"},
                        },
                        {
                            "comparison":  "equals",
                            "field":       "responsible",
                            "contentType": "FilterTermRef",
                            "value": [...]megaplan.QueryParams{
                                {"id": 1000129, "contentType": "Employee"},
                            },
                        },
                        {
                            "comparison":  "equals",
                            "field":       "statusChangeTime",
                            "contentType": "FilterTermDate",
                            "value": megaplan.QueryParams{
                                "contentType": "IntervalDates",
                                "from":        megaplan.CreateEnity("DateOnly", from),
                                "to": megaplan.QueryParams{
                                    "contentType": "DateOnly",
                                    "year":        now.Year(),
                                    "month":       int(now.Month()) - 1,
                                    "day":         now.Day(),
                                },
                            },
                        },
                    },
                },
            },
        }
        searchParams["filter"] = filterParams

        {
            // вариант отправки через DoRequestAPI, внутри формируется корректный http.Request, если http.Response был сжат, то будет разархивирован
            rc, err := c.DoRequestAPI(http.MethodGet, endpoint, searchParams, nil)
            if err != nil {
                panic(err)
            }
            defer rc.Close()
            os.Stdout.ReadFrom(rc)
        }
        {
            // пример с использование Do, внучную собирается http.Request (добавляются необходимые заголовки, http.Response никак не обрабатывается перед возвратом)
            c.SetOptions(megaplan.OptionEnableAcceptEncodingGzip(false))
            request, err := http.NewRequest(http.MethodGet, domain, nil)
            if err != nil {
                panic(err)
            }
            request.URL.Path = endpoint
            request.URL.RawQuery = searchParams.QueryEscape() // параметры будут правильно экранированы
            response, err := c.Do(request)
            if err != nil {
                panic(err)
            }
            defer response.Body.Close()
            os.Stdout.ReadFrom(response.Body)
        }
    }
```

## Чтение ответа
С появлением дженериков улучшена функция для чтения ответов от api.
Внутри функции есть проверка на Content-Type, если это не json, то в 99% это html с ошибкой. В этом случае функция вернет ошибку с текстом в виде html строки.


```golang
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stvoidit/megaplan/v3"
)

const (
	DOMAIN      = `https://example.ru`
	TOKEN       = `TOKEN`
	ACCOUNTINFO = `/api/v3/accountInfo`
)

type AccountInfo struct {
	ID                string         `json:"id"`
	ContentType       string         `json:"contentType"`
	PermanentHost     string         `json:"permanentHost"`
	AccountName       string         `json:"accountName"`
	BuildVersion      string         `json:"buildVersion"`
	SystemProductName string         `json:"systemProductName"`
	TarifId           string         `json:"tarifId"`
	LicenceEndDate    map[string]any `json:"licenceEndDate"`
	MobileEndDate     map[string]any `json:"mobileEndDate"`
	PaidToDate        map[string]any `json:"paidToDate"`
	LicenseExpired    bool           `json:"licenseExpired"`
	MegamailDomain    string         `json:"megamailDomain"`
	DaysRemaining     int            `json:"daysRemaining"`
	TimeCreated       string         `json:"timeCreated"`
}

func (ai AccountInfo) String() string {
	var sb strings.Builder
	e := json.NewEncoder(&sb)
	e.SetIndent("", "  ")
	e.Encode(&ai)
	return sb.String()
}

func main() {
	c := megaplan.NewClient(DOMAIN, TOKEN,
		megaplan.OptionEnableAcceptEncodingGzip(true),
		megaplan.OptionInsecureSkipVerify(true))
	res, err := c.DoRequestAPI(http.MethodGet, ACCOUNTINFO, nil, nil)
	if err != nil {
		panic(err)
	}
    // Вы можете указать типа как "any", если вам нужно стандартное поведение json.Decode - возврат в виде map[string]any
	body, err := megaplan.ParseResponse[AccountInfo](res)
	if err != nil {
		panic(err)
	}
	fmt.Println(body)
}
```


## __!__ Про типы и сущности "мегаплана" __!__

\* _не актуально с появлением дженериков_

Многие реализации библиотек для API "Мегаплана" пытаются строго типизировать и описать полностью сущности, которыми оперирует "Мегаплан".
Однако это подход влечет за собой обязанность этих библиотек поддерживать согласованность с версиями "Мегаплана", а так же каким-то образом поддерживать кастомные варианты полей.
Данная библиотека является просто оберткой для использования API v3 и включает минимальное кол-во вспомогательных функций для составления запросов и парсинга ответов.

В силу специфики строения сущностей "Мегаплана" некоторые типы могут некорретно собираться функцией __megaplan.BuildQueryParams__, поэтому выше даны примере, как можно "дособрать" необходимые объекты.
