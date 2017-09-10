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

    PLAYER_WIDTH = 15
    PLAYER_HEIGHT = 80
    PLAYER_PADDING = 15
    PLAYER_SPEED = 6
    BALL_SIZE = 15
    WINDOW_WIDTH = 800
    WINDOW_HEIGHT = 600
    MIN_UPDATE_PERIOD = 16*time.Millisecond
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
    incomming_events net.Conn
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

        go func(event_connection net.Conn){
            gamestate_connection,err:=net.Dial("udp4", connection.RemoteAddr().String())
            if err!=nil{
                log.Println("[accept_incomming_connections] Error (net.Dial):", err)
                return
            }

            c<-PlayersConnections{event_connection, gamestate_connection}
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

func forward_event_messages(message_sender MessageSender, incomming_events net.Conn, finished *int64){
    defer incomming_events.Close()
    buffer:=make([]byte, 100)
    for atomic.LoadInt64(finished)==0{
        n,err:=incomming_events.Read(buffer)
        if err!=nil{
            log.Println("[forward_event_messages] Error (connection.Read):",  incomming_events.RemoteAddr(), err)
            if err.Error()=="EOF"{
                atomic.StoreInt64(finished, 1)
            }
            continue
        }

        if n!=2{
            log.Println("[forward_event_messages] n!=2", buffer)
            continue
        }

        message_sender.send((uint16(buffer[1])<<8) | (uint16(buffer[0])<<0))
    }

    log.Println("[forward_event_messages] Connection closed")
}

type Keystate struct{
    up bool
    down bool
}

type Player struct{
    keystate Keystate
    pos int16
}

func (p *Player) update_keystate(message uint16){
    if message==2{
        p.keystate.up=true
    } else if message==3{
        p.keystate.up=false
    } else if message==4{
        p.keystate.down=true
    } else if message==5{
        p.keystate.down=false
    }
}

func (p *Player) move(){
    if (p.keystate.up && p.keystate.down) {
        return
    }

    if p.keystate.up{
        p.pos-=PLAYER_SPEED
        if p.pos<0{
            p.pos=0
        }
    }

    if p.keystate.down{
        p.pos+=PLAYER_SPEED
        if p.pos+PLAYER_HEIGHT>=WINDOW_HEIGHT{
            p.pos=WINDOW_HEIGHT-PLAYER_HEIGHT-1
        }
    }
}

func fill_send_buffer(buffer []byte, p1pos, p2pos int16){
    buffer[0]=byte((p1pos>>0)&0xFF)
    buffer[1]=byte((p1pos>>8)&0xFF)
    buffer[2]=byte((p2pos>>0)&0xFF)
    buffer[3]=byte((p2pos>>8)&0xFF)
}

func run_game(player1s_connections, player2s_connections PlayersConnections){
    player1s_outgoing_gamestate:=player1s_connections.outgoing_gamestate
    player2s_outgoing_gamestate:=player2s_connections.outgoing_gamestate
    defer player1s_outgoing_gamestate.Close()
    defer player2s_outgoing_gamestate.Close()

    p1_message_sender, p1_message_receiver:=create_message_transmission_pair()
    p2_message_sender, p2_message_receiver:=create_message_transmission_pair()
    finished:=new(int64)
    go forward_event_messages(p1_message_sender, player1s_connections.incomming_events, finished)
    go forward_event_messages(p2_message_sender, player2s_connections.incomming_events, finished)

    player1:=Player{}
    player2:=Player{}

    send_buffer:=[8]byte{}
    now:=time.Now()
    outer_loop: for atomic.LoadInt64(finished)==0{
        for _,message:=range p1_message_receiver.receive(){
            if message==1{
                atomic.StoreInt64(finished, 1)
                break outer_loop
            } else {
                player1.update_keystate(message)
            }
        }

        for _,message:=range p2_message_receiver.receive(){
            if message==1{
                atomic.StoreInt64(finished, 1)
                break outer_loop
            } else {
                player2.update_keystate(message)
            }
        }

        player1.move()
        player2.move()
        fill_send_buffer(send_buffer[:], player1.pos, player2.pos)

        iterations_duration:=time.Since(now)
        // log.Println(iterations_duration)
        if iterations_duration<MIN_UPDATE_PERIOD{
            time.Sleep(MIN_UPDATE_PERIOD-iterations_duration)
        }

        n,err:=player1s_outgoing_gamestate.Write(send_buffer[:])
        if err!=nil{
            log.Println("[run_game] Error (player1s_outgoing_gamestate.Write):", n, err)
        }
        n,err=player2s_outgoing_gamestate.Write(send_buffer[:])
        if err!=nil{
            log.Println("[run_game] Error (player2s_outgoing_gamestate.Write):", n, err)
        }

        now=time.Now()
    }
}

// TODO: experiment with lower buffer sizes to find errors/performance costs
// TODO: send gamestate in each loop, even if no changes where made
// TODO: analyze "framerate" (need optimizing?)
// TODO: REMOVE MAGIC NUMBERS
func main() {
    c:=make(chan PlayersConnections, 5)
    go find_server_protocol_server()
    go accept_incomming_connections(c)

    for{
        player1s_connections:=<-c
        player2s_connections:=<-c
        go run_game(player1s_connections, player2s_connections)
    }
}