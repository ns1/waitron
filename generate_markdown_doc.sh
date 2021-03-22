go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/go-swagger/go-swagger

go build -o $GOPATH/bin/ $GOPATH/src/github.com/go-swagger/go-swagger/cmd/swagger
$GOPATH/bin/swag init
$GOPATH/bin/swagger generate markdown -f docs/swagger.json --output API.md
mv api.md API.md 2>/dev/null
