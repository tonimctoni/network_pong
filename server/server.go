package main;

import "net"
import "log"
import "strings"
import "errors"
import "sync/atomic"
import "time"

const(
    find_server_protocol_listen_address = ":1234"
    message_start = "cB9"
    looking_for_server_message = "LFS"
    looking_for_server_response = "IAS"
    // character_whitelist = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    connect_address = ":1235"
    message_buffer_size = 1000
)

// func has_whitelisted_chars_only(s string) bool {
//     for _,character:=range s{
//         if !strings.ContainsAny(string(character), character_whitelist){
//             return false
//         }
//     }
//     return true
// }

func get_ip_and_port_from_address(address string) (string, string, error) {
    i:=strings.LastIndex(address, ":")
    if i<0 {
        return "", "", errors.New("Weird address")
    }
    return address[:i], address[i+1:], nil
}

func find_server_protocol_server() {
    packet_connection, err:=net.ListenPacket("udp4", find_server_protocol_listen_address)
    if err!=nil{
        log.Fatalln("[find_server_protocol_server] Error (net.ListenPacket):", err)
    }
    defer packet_connection.Close()
    log.Println("[find_server_protocol_server] Listening on:", packet_connection.LocalAddr())

    for{
        buffer:=make([]byte, 100)
        n, addr, err:=packet_connection.ReadFrom(buffer)
        if err!=nil{
            log.Println("[find_server_protocol_server] Error (packet_connection.ReadFrom):", err)
            continue
        }

        if n==len(message_start+looking_for_server_message) && string(buffer[:n])==message_start+looking_for_server_message{
            go func(addr string, buffer []byte){
                connection,err:=net.Dial("udp4", addr)
                if err!=nil{
                    log.Println("[find_server_protocol_server] Error (net.Dial):", err)
                    return
                }
                defer connection.Close()

                time.Sleep(100*time.Millisecond)
                _,err=connection.Write([]byte(message_start+looking_for_server_response))
                if err!=nil{
                    log.Println("[find_server_protocol_server] Error (connection.Write):", err)
                    return
                }

                log.Println("[find_server_protocol_server] find_server_protocol response sent to:", addr)
            }(addr.String(), buffer)
        } else {
            log.Println("[find_server_protocol_server] Error: invalid request_from:", addr.String())
        }
    }
}

type PlayersConnections struct{
    incomming_keystrokes net.Conn
    outgoing_gamestate net.Conn
}

func accept_incomming_connections(c chan PlayersConnections) {
    listener,err:=net.Listen("tcp4", connect_address)
    if err!=nil{
        log.Fatalln("[accept_incomming_connections] Error (net.Listen):", err)
    }
    defer listener.Close()
    log.Println("[accept_incomming_connections] Listening on:", listener.Addr())

    for{
        connection, err:=listener.Accept()
        if err != nil {
            log.Println("Error (listener.Accept):", err)
            continue
        }

        go func(keystroke_connection net.Conn){
            gamestate_connection,err:=net.Dial("udp4", connection.RemoteAddr().String())
            if err!=nil{
                log.Println("[accept_incomming_connections] Error (net.Dial):", err)
                return
            }

            c<-PlayersConnections{keystroke_connection, gamestate_connection}
        }(connection)
    }
}

type MessageBuffer struct{
    pos int64
    buffer [message_buffer_size]uint16
}

type MessageSender struct{
    message_buffer *MessageBuffer
    mb_sender chan *MessageBuffer
}

type MessageReceiver struct{
    message_buffer *MessageBuffer
    mb_receiver chan *MessageBuffer
    pos int64
}

func create_message_transmission_pair() (MessageSender, MessageReceiver){
    message_buffer_ptr:=&MessageBuffer{}
    mbc:=make(chan *MessageBuffer, 2)

    return MessageSender{message_buffer_ptr, mbc}, MessageReceiver{message_buffer_ptr, mbc, 0}
}

func (m *MessageSender) send(message uint16){
    pos:=atomic.LoadInt64(&m.message_buffer.pos)

    m.message_buffer.buffer[pos]=message
    atomic.AddInt64(&m.message_buffer.pos, 1)
    pos+=1

    if pos>message_buffer_size{
        panic("pos>message_buffer_size")
    }

    if pos==message_buffer_size{
        new_message_buffer_ptr:=&MessageBuffer{}
        m.message_buffer=new_message_buffer_ptr
        m.mb_sender<-new_message_buffer_ptr
    }
}

func (m *MessageReceiver) receive() []uint16{
    if m.pos>message_buffer_size{
        panic("m.pos>message_buffer_size")
    }

    if m.pos==message_buffer_size{
        m.message_buffer=<-m.mb_receiver
        m.pos=0
    }

    pos:=atomic.LoadInt64(&m.message_buffer.pos)
    to_return:=m.message_buffer.buffer[m.pos:pos]
    m.pos=pos
    return to_return
}

func forward_keystroke_messages(message_sender MessageSender, incomming_keystrokes net.Conn, finished *int64){
    defer incomming_keystrokes.Close()
    buffer:=make([]byte, 100)
    for atomic.LoadInt64(finished)==0{
        n,err:=incomming_keystrokes.Read(buffer)
        if err!=nil{
            log.Println("[forward_keystroke_messages] Error (connection.Read):",  incomming_keystrokes.RemoteAddr(), err)
            if err.Error()=="EOF"{
                atomic.StoreInt64(finished, 1)
            }
            continue
        }

        if n!=2{
            log.Println("[forward_keystroke_messages] n!=2", buffer)
            continue
        }

        message_sender.send((uint16(buffer[1])<<8) | (uint16(buffer[0])<<0))
    }

    log.Println("[forward_keystroke_messages] Connection closed")
}

func start_game(player1s_connections, player2s_connections PlayersConnections){
    player1s_outgoing_gamestate:=player1s_connections.outgoing_gamestate
    player2s_outgoing_gamestate:=player2s_connections.outgoing_gamestate
    defer player1s_outgoing_gamestate.Close()
    defer player2s_outgoing_gamestate.Close()

    p1_message_sender, p1_message_receiver:=create_message_transmission_pair()
    p2_message_sender, p2_message_receiver:=create_message_transmission_pair()
    finished:=new(int64)
    go forward_keystroke_messages(p1_message_sender, player1s_connections.incomming_keystrokes, finished)
    go forward_keystroke_messages(p2_message_sender, player2s_connections.incomming_keystrokes, finished)

    now:=time.Now()
    for atomic.LoadInt64(finished)==0{
        log.Println("p1:", p1_message_receiver.receive())
        log.Println("p2:", p2_message_receiver.receive())

        log.Println(time.Since(now))
        now=time.Now()
    }
}

//TODO: experiment with lower buffer sizes to find errors/performance costs
func main() {
    c:=make(chan PlayersConnections, 5)
    go find_server_protocol_server()
    go accept_incomming_connections(c)

    for{
        player1s_connections:=<-c
        player2s_connections:=<-c
        start_game(player1s_connections, player2s_connections)
    }
}