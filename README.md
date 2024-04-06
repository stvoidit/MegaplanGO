Реализованы 1 и 3 версии API. Каждая из реализаций для версий находится в своей ветке.
По умолчанию базовой веткой остается v1, т.к. в собственных проектах мне необходима аутентификация пользователей, которая не реализована в v3 самого мегаплана, для сторонних приложений.

## v1

https://dev.megaplan.ru/r1905/api/index.html

Представляет простую обертку над http методами GET и POST
Алгоритм шифрования запроса см. в методе __queryHashing__, может быть реализован на любом ЯП.


Установка:
```
go get github.com/stvoidit/megaplan
```

Пример использования:
```golang
var api = megaplan.NewAPI(accessID, secretKey, myhost)
response, err := api.GET("/BumsCommonApiV01/UserInfo/id.api", nil)
if err != nil {
	panic(err)
}
defer response.Body.Close()
fmt.Println(response.Status)
type UserInfo struct {
	UserID       int64  `json:"UserId"`
	EmployeeID   int64  `json:"EmployeeId"`
	ContractorID string `json:"ContractorId"`
}
var user = new(UserInfo)
if err := json.NewDecoder(response.Body).Decode(megaplan.ExpectedResponse(user)); err != nil {
	panic(err)
}
fmt.Printf("%+v\n", user)
```

## v3

https://dev.megaplan.ru/r1905/apiv3/index.html

Реализация для APIv3 находится в отдельной ветке [https://github.com/stvoidit/megaplan/tree/v3](https://github.com/stvoidit/megaplan/tree/v3).

Описание и примеры там же.

Установка:
```
go get github.com/stvoidit/megaplan/v3
```

## Примечание

v1 и v3 координально отличаются по схемам данных, обработке и содержанию.
Многие сущности не описаны в [v3 документации](https://demo.megaplan.ru/api/v3/docs).
Например имеется endpoint на __/api/v3/department__, но при этом сущность сотрудников вообще не имеет данных об отделах.
Имеется endpoint __/api/v3/position__, но не описан в докуменатации, хотя представляет структурированные данные о должностях сотрудников.
