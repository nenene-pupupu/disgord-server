# disGOrd

## How to Run
1. check if `go` is installed
```
go version
```

2. get packages and generate codes
```
go get
go generate ./ent
```

3. create `.env` file and set `SECRET`
```
cp example.env .env

# or generate random string
echo "SECRET=$(openssl rand -hex 8)" > .env
```

4. run server
```
go build
./disgord

# or
go run main.go
```

5. access through `localhost:8080`

6. check APIs at `/swagger/index.html`
