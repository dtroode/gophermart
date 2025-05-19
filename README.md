# go-musthave-diploma-tpl

## генерация sqlc
```
go tool sqlc generate
```

## генерация swagger
```
go tool swag init -g cmd/gophermart/main.go -d .,./internal/application,./internal/api/http
```

## генерация моков
```
docker run -v "$PWD":/src -w /src vektra/mockery --all
```
