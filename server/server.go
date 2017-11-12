package main;
// import "net"
// import "log"
// import "strings"
// import "errors"
// import "sync/atomic"
// import "time"
// import "fmt"


// TODO: send gamestate in each loop, even if no changes where made
// TODO: analyze "framerate" (need optimizing?)
// TODO: REMOVE MAGIC NUMBERS
// TODO: Add timeout to connections
func main() {
    c:=make(chan PlayerConnection, 100)
    go find_server_service()
    go accept_incomming_connections(c)

    for{
        player1_connection:=<-c
        player2_connection:=<-c
        go run_game(player1_connection, player2_connection)
    }
}