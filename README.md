# disGOrd

## How to Run
1. check if `go` is installed
    ```sh
    go version
    ```

1. execute shell script `init.sh` for initial settings
    ```sh
    ./init.sh
    ```

1. run server
    ```sh
    go run main.go
    ```

1. access through `localhost:8080`

1. check APIs at `/swagger/index.html`

1. after each git pull, run `go generate ./...`

## It supports
- real-time text chat with multiple clients through WebSocket
- SFU media server for real-time voice/video chat
- JWT user authorization based on Refresh Token Rotation
- public/private chatroom
- previous chat history of the chatroom

## It uses
- [gin-gonic/gin](https://github.com/gin-gonic/gin): HTTP web framework written in Go
- [gorilla/websocket](https://github.com/gorilla/websocket): WebSocket implementation for Go
- [pion/webrtc](https://github.com/pion/webrtc): Pure Go implementation of the WebRTC API
- [ent/ent](https://github.com/ent/ent): Simple, yet powerful ORM
- [mattn/go-sqlite3](https://github.com/mattn/go-sqlite3): sqlite3 driver for go
- [golang-jwt/jwt](https://github.com/golang-jwt/jwt): Golang implementation of JSON Web Tokens (JWT)
- [swaggo/swag](https://github.com/swaggo/swag): RESTful API documentation with Swagger 2.0 for Go
