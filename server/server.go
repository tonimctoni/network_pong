package main;
// import "net"
// import "log"
// import "strings"
// import "errors"
// import "sync/atomic"
// import "time"
// import "fmt"


// TODO: experiment with lower buffer sizes to find errors/performance costs
// TODO: send gamestate in each loop, even if no changes where made
// TODO: analyze "framerate" (need optimizing?)
// TODO: REMOVE MAGIC NUMBERS
// TODO: Add timeout to connections
func main() {
    // Buffer size needs to be one for synchronization
    c:=make(chan PlayerConnection)
    cn:=make(chan byte)
    go find_server_service()
    go accept_incomming_connections(c, cn)

    for{
        cn<-1
        player1_connection:=<-c
        cn<-2
        player2_connection:=<-c
        go run_game(player1_connection, player2_connection)
    }
}