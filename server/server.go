package main;

import "net"
import "log"
import "strings"
import "errors"
import "sync/atomic"
import "time"
import "fmt"

const(
    find_server_protocol_listen_address = ":1234"
    message_start = "cB9"
    looking_for_server_message = "LFS"
    looking_for_server_response = "IAS"
    connect_to_server_message = "CTS"
    // character_whitelist = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    connect_address = ":1235"
    message_buffer_size = 1000
    port_counter_start = 10002
    // max_event_messages_to_process = 2

    PLAYER_WIDTH = 15
    PLAYER_HEIGHT = 80
    PLAYER_PADDING = 15
    PLAYER_SPEED = 6
    BALL_SIZE = 15
    BALL_SPEED = 6
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

func uint16_from_slice(slice []byte) uint16{
    return (uint16(slice[1])<<8) | (uint16(slice[0])<<0)
}

func uint16_to_slice(n uint16, slice []byte){
    slice[0]=byte((n>>0)&0xFF)
    slice[1]=byte((n>>8)&0xFF)
}

func int16_to_slice(n int16, slice []byte){
    slice[0]=byte((n>>0)&0xFF)
    slice[1]=byte((n>>8)&0xFF)
}

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

type PlayerConnection struct{
    gamestate_port uint16
    eventstate_port uint16
    ip_address string
}

// type TowPlayerConnections{
//     p1 PlayerConnection
//     p2 PlayerConnection
// }

func accept_incomming_connections(pcc chan PlayerConnection, pn chan byte){
    listener,err:=net.Listen("tcp4", connect_address)
    if err!=nil{
        log.Fatalln("[accept_incomming_connections] Error (net.Listen):", err)
    }
    defer listener.Close()
    log.Println("[accept_incomming_connections] Listening on:", listener.Addr())

    port_counter:=uint16(port_counter_start)
    for{
        connection, err:=listener.Accept()
        if err != nil {
            log.Println("[accept_incomming_connections] Error (listener.Accept):", err)
            continue
        }

        port_counter+=2
        if port_counter<10{
            port_counter=port_counter_start
        }
        go func(connection net.Conn, gamestate_port, eventstate_port uint16){
            defer connection.Close()
            buffer:=make([]byte, 100)
            n,err:=connection.Read(buffer)
            if err!=nil{
                log.Println("[accept_incomming_connections] Error (connection.Read):", err)
                return
            }

            if n!=len(message_start+connect_to_server_message) || string(buffer[:n])!=message_start+connect_to_server_message{
                log.Println("[accept_incomming_connections] Error: Incorrect connect message from:", connection.RemoteAddr())
                return
            }

            ip_address,_,err:=get_ip_and_port_from_address(connection.RemoteAddr().String())
            if err!=nil{
                log.Println("[accept_incomming_connections] Error (get_ip_and_port_from_address):", err)
                return
            }

            buffer[0]=<-pn
            uint16_to_slice(gamestate_port, buffer[1:3])
            uint16_to_slice(eventstate_port, buffer[3:5])
            _,err=connection.Write(buffer[:5])
            if err!=nil{
                log.Println("[accept_incomming_connections] Error (connection.Write):", err)
                pn<-buffer[0]
                return
            }
            log.Println("[accept_incomming_connections] Connection established for:", connection.RemoteAddr())

            pcc<-PlayerConnection{gamestate_port, eventstate_port, ip_address}
        }(connection, port_counter, port_counter+1)
    }
}

type MessageBuffer struct{
    pos int64
    buffer [message_buffer_size][2]uint16
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

func (m *MessageSender) send(message [2]uint16){
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

func (m *MessageReceiver) receive() [][2]uint16{
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

func forward_eventstate_messages(message_sender MessageSender, player_connection PlayerConnection, finished *int64){
    defer atomic.StoreInt64(finished, 1)

    eventstate_connection, err:=net.ListenPacket("udp4", fmt.Sprintf("%s:%d", player_connection.ip_address, player_connection.eventstate_port))
    if err!=nil{
        log.Println("[forward_eventstate_messages] Error (net.ListenPacket):", err)
        return
    }
    // log.Println("[forward_eventstate_messages] Listening to events from:", player_connection.ip_address)
    // defer log.Println("[forward_eventstate_messages] Connection closed for:", player_connection.ip_address)
    defer eventstate_connection.Close()


    buffer:=make([]byte, 100)
    for atomic.LoadInt64(finished)==0{
        n,addr,err:=eventstate_connection.ReadFrom(buffer)
        if err!=nil{
            log.Println("[forward_eventstate_messages] Error (eventstate_connection.Read):", addr, err)
            if err.Error()=="EOF"{
                return
            }
        }

        if n!=4{
            log.Println("[forward_eventstate_messages] n!=4", buffer)
            continue
        }

        message_sender.send([2]uint16{uint16_from_slice(buffer[0:2]), uint16_from_slice(buffer[2:4])})
    }
}

type Keystate struct{
    up bool
    down bool
}

type Player struct{
    keystate Keystate
    last_iteration uint16
    pos int16
}

func (p *Player) update_keystate(eventstate uint16, iteration uint16){
    if eventstate&2!=0{
        p.keystate.up=true
    } else {
        p.keystate.up=false
    }

    if eventstate&4!=0{
        p.keystate.down=true
    } else {
        p.keystate.down=false
    }

    p.last_iteration=iteration
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

type Ball struct{
    pos_x int16
    pos_y int16
    speed_x int16
    speed_y int16
}

func (b *Ball) move_y(){
    b.pos_y+=b.speed_y
    if b.pos_y<0{
        b.pos_y=-b.pos_y
        b.speed_y=-b.speed_y
    }
    if b.pos_y+BALL_SIZE>=WINDOW_HEIGHT{
        b.pos_y=(WINDOW_HEIGHT-BALL_SIZE-1)-(b.pos_y+BALL_SIZE-WINDOW_HEIGHT)
        b.speed_y=-b.speed_y
    }
}

func (b *Ball) move(p1pos, p2pos int16) int{
    if b.pos_x+b.speed_x<PLAYER_PADDING+PLAYER_WIDTH && b.pos_x>PLAYER_PADDING+PLAYER_WIDTH{
        portion_moved:=float64(PLAYER_PADDING+PLAYER_WIDTH-b.pos_x)/float64(b.speed_x)
        this_speed_x:=int16(portion_moved*float64(b.speed_x))
        this_speed_y:=int16(portion_moved*float64(b.speed_y))

        moved_portion_y:=b.pos_y+this_speed_y
        if p1pos<=moved_portion_y+BALL_SIZE && p1pos+PLAYER_HEIGHT>=moved_portion_y{
            b.pos_x+=this_speed_x-(b.speed_x-this_speed_x)
            b.speed_x=-b.speed_x
            b.move_y()
            return 0
        } else {
            b.pos_x=WINDOW_WIDTH/2-BALL_SIZE/2
            b.pos_y=WINDOW_HEIGHT/2-BALL_SIZE/2
            b.speed_x=BALL_SPEED
            b.speed_y=-BALL_SPEED
            return 2
        }
    } else if b.pos_x+b.speed_x>WINDOW_WIDTH-(PLAYER_PADDING+PLAYER_WIDTH) && b.pos_x<WINDOW_WIDTH-(PLAYER_PADDING+PLAYER_WIDTH){
        b.pos_x-=b.speed_x
        b.speed_x=-b.speed_x
        b.move_y()
        return 0
    } else {
        b.pos_x+=b.speed_x
        b.move_y()
        return 0
    }
}

func run_game(player1_connection, player2_connection PlayerConnection) {
    finished:=new(int64)
    defer atomic.StoreInt64(finished, 1)
    p1_message_sender, p1_message_receiver:=create_message_transmission_pair()
    p2_message_sender, p2_message_receiver:=create_message_transmission_pair()
    go forward_eventstate_messages(p1_message_sender, player1_connection, finished)
    go forward_eventstate_messages(p2_message_sender, player2_connection, finished)

    p1_gamestate_connection,err:=net.Dial("udp4", fmt.Sprintf("%s:%d", player1_connection.ip_address, player1_connection.gamestate_port))
    if err!=nil{
        log.Println("[run_game] Error (net.Dial):", err)
        return
    }
    defer p1_gamestate_connection.Close()

    p2_gamestate_connection,err:=net.Dial("udp4", fmt.Sprintf("%s:%d", player2_connection.ip_address, player2_connection.gamestate_port))
    if err!=nil{
        log.Println("[run_game] Error (net.Dial):", err)
        return
    }
    defer p2_gamestate_connection.Close()

    player1:=Player{}
    player2:=Player{}
    ball:=Ball{WINDOW_WIDTH/2-BALL_SIZE/2, WINDOW_HEIGHT/2-BALL_SIZE/2, BALL_SPEED, -BALL_SPEED}

    now:=time.Now()
    for atomic.LoadInt64(finished)==0{
        p1_messages:=p1_message_receiver.receive()
        if len(p1_messages)!=0{
            player1.update_keystate(p1_messages[len(p1_messages)-1][1], p1_messages[len(p1_messages)-1][0])
        }

        p2_messages:=p2_message_receiver.receive()
        if len(p2_messages)!=0{
            player2.update_keystate(p2_messages[len(p2_messages)-1][1], p2_messages[len(p2_messages)-1][0])
        }

        player1.move()
        player2.move()
        ball.move(player1.pos, player2.pos)

        // if len(p1_messages)>max_event_messages_to_process{
        //     p1_messages=p1_messages[len(p1_messages)-max_event_messages_to_process:]
        // }
        // for _,message:=range p1_messages{
        //     if message[1]&1!=0{
        //         return
        //     }
        //     player1.update_keystate(message[1], message[0])
        //     player1.move()
        // }

        // if len(p2_messages)>max_event_messages_to_process{
        //     p2_messages=p2_messages[len(p2_messages)-max_event_messages_to_process:]
        // }
        // for _,message:=range p2_messages{
        //     if message[1]&1!=0{
        //         return
        //     }
        //     player2.update_keystate(message[1], message[0])
        //     player2.move()
        // }

        send_buffer:=[10]byte{}
        int16_to_slice(player1.pos, send_buffer[2:4])
        int16_to_slice(player2.pos, send_buffer[4:6])
        int16_to_slice(ball.pos_x, send_buffer[6:8])
        int16_to_slice(ball.pos_y, send_buffer[8:10])

        uint16_to_slice(player1.last_iteration, send_buffer[0:2])
        n,err:=p1_gamestate_connection.Write(send_buffer[:])
        if err!=nil{
            log.Println("[run_game] Error (p1_gamestate_connection.Write):", n, err)
            // if strings.Contains(err.Error(), "connection refused"){
            //     return
            // }
        }
        uint16_to_slice(player2.last_iteration, send_buffer[0:2])
        n,err=p2_gamestate_connection.Write(send_buffer[:])
        if err!=nil{
            log.Println("[run_game] Error (p2_gamestate_connection.Write):", n, err)
            // if strings.Contains(err.Error(), "connection refused"){
            //     return
            // }
        }

        iteration_duration:=time.Since(now)
        // log.Println(iteration_duration)
        if iteration_duration<MIN_UPDATE_PERIOD{
            time.Sleep(MIN_UPDATE_PERIOD-iteration_duration)
        }

        now=time.Now()
    }

}

// TODO: experiment with lower buffer sizes to find errors/performance costs
// TODO: send gamestate in each loop, even if no changes where made
// TODO: analyze "framerate" (need optimizing?)
// TODO: REMOVE MAGIC NUMBERS
// TODO: Add timeout to connections
func main() {
    // Buffer size needs to be one for synchronization
    c:=make(chan PlayerConnection)
    cn:=make(chan byte)
    go find_server_protocol_server()
    go accept_incomming_connections(c, cn)

    for{
        cn<-1
        player1_connection:=<-c
        cn<-2
        player2_connection:=<-c
        go run_game(player1_connection, player2_connection)
    }
}