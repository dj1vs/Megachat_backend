# Megachat_backend
Бэкенд для МегаЧата

## Отправка сообщений с фронта
- Сообщения с фронта на бэк приходят по адресу `ws://127.0.0.1:8800/ws`
- Время принимается в формате `2024-02-18T12:34:56Z`
- Ответ фронту на его запрос приходит после определения статуса отправки сообщений на сервис кодирования (ну или раньше, если например json был неправильно сформирован)
- Если json запроса с фронта неправильно сформирован, в ответном запросе `username` устанавливается как пустая строка, а `time` как время отправки ответа 
- Текущий лимит на отправку сообщений с фронта: 1024 байта (просто заглушка)


## Отправка сообщений на сервис кодирования
- Сообщения отправляются через POST запросы
- Нумерация сегментов - с нуля
- Если размер сообщения в байтах не кратен 140, остаток отсылается последним сегментом.
- Строка с текстом закодирована в Base64 (она так кодируется по дефолту в go)

## Получение сообщений от сервиса кодирования
- Строка с текстом закодирована в Base64