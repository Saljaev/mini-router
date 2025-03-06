# Описание
Пакет предоставляет собой минималистичный роутер, а также [обертку](router/context.go) контекста, для упрощения работы с handlers.

Роутер поддерживает возможность создания [групп](#https://github.com/Saljaev/mini-router/blob/master/router/router.go#L115-l127) 
роутеров, для маршрутизации запросов с одинаковым префиксом.

А также простой синтаксис и обработку для handler и middleware.

Есть [враппер](https://github.com/Saljaev/mini-router/blob/master/router/router.go#L36-L40) для работы с обычными функциями
вида ```func (w http.ResponseWriter, r *http.Request)```.

*Два способа использования handlers и middleware:*
1. Передача handlers и middleware в правильной последовательности в функции ```POST```/```GET```
2. Использование команды ```Use```, которая добавляет middleware (работает на алгоритме **FIFO**),
который будет выполняться перед handlers

## Контекст
Содержит в себе [slog](#https://pkg.go.dev/log/slog) для логирования следующих уровней: 
- Debug
- Error
- Info

*Основной функционал:*
- ```SuccessWithData``` - запись ответа со статусом 200
- ```WriteFailure``` - запись ответа с указанным кодом и ошибкой
- ```WithTimeout``` - установка timeout на текущий контекст
- ```Set``` и ```Value``` - запись и чтение данных с контекста
- ```ctx.GetFromQuery()``` - получение данных из query только из **GET**
- ```ctx.GetFromPath()```- получение данных из path только из **GET**
- ```ctx.Decode()``` - парсинг тела запроса только из **POST**
- ```ctx.GetFromHeader()``` - получение данных из заголовка 

