TODO: Заполнить README

```
mikhail@HighLoadHT1:~/HL_homework_1$ curl -s localhost:8080/handlers | jq
{
  "handlers": [
    {
      "method": "POST",
      "path": "/items",
      "handler": "POST /items"
    },
    {
      "method": "GET",
      "path": "/items/{id}",
      "handler": "GET /items/{id}"
    },
    {
      "method": "PUT",
      "path": "/items/{id}",
      "handler": "PUT /items/{id}"
    }
  ]
}
```
