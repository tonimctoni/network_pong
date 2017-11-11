package main;
import "net"
import "log"
import "sync/atomic"
import "time"
import "fmt"



func process_eventstate_messages(eventstate *uint32, player_connection PlayerConnection, finished *int32){
    defer atomic.StoreInt32(finished, 1)

    eventstate_connection, err:=net.ListenPacket("udp4", fmt.Sprintf("%s:%d", player_connection.ip_address, player_connection.eventstate_port))
    if err!=nil{
        log.Println("[process_eventstate_messages] Error (net.ListenPacket):", err)
        return
    }
    defer eventstate_connection.Close()

    buffer:=make([]byte, 100)
    for atomic.LoadInt32(finished)==0{
        n,addr,err:=eventstate_connection.ReadFrom(buffer)
        if err!=nil{
            log.Println("[process_eventstate_messages] Error (eventstate_connection.Read):", addr, err)
            if err.Error()=="EOF"{
                return
            }
        }

        if n!=4{
            log.Println("[process_eventstate_messages] n!=4", buffer)
            continue
        }

        atomic.StoreUint32(eventstate, uint32_from_slice(buffer[0:4]))
    }
}

func run_game(player1_connection, player2_connection PlayerConnection) {
    finished:=new(int32)
    defer atomic.StoreInt32(finished, 1)
    p1_eventstate:=new(uint32)
    p2_eventstate:=new(uint32)
    go process_eventstate_messages(p1_eventstate, player1_connection, finished)
    go process_eventstate_messages(p2_eventstate, player2_connection, finished)

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
    for atomic.LoadInt32(finished)==0{
        player1.update_keystate(atomic.LoadUint32(p1_eventstate))
        player2.update_keystate(atomic.LoadUint32(p2_eventstate))

        player1.move()
        player2.move()
        ball.move(player1.pos, player2.pos)

        send_buffer:=[16]byte{}
        int16_to_slice(player1.pos, send_buffer[0:2])
        int16_to_slice(player2.pos, send_buffer[2:4])
        int16_to_slice(ball.pos_x, send_buffer[4:6])
        int16_to_slice(ball.pos_y, send_buffer[6:8])
        int16_to_slice(ball.speed_x, send_buffer[8:10])
        int16_to_slice(ball.speed_y, send_buffer[10:12])
        int16_to_slice(0, send_buffer[12:14])
        int16_to_slice(0, send_buffer[14:16])

        n,err:=p1_gamestate_connection.Write(send_buffer[:])
        if err!=nil{
            log.Println("[run_game] Error (p1_gamestate_connection.Write):", n, err)
        }

        n,err=p2_gamestate_connection.Write(send_buffer[:])
        if err!=nil{
            log.Println("[run_game] Error (p2_gamestate_connection.Write):", n, err)
        }

        iteration_duration:=time.Since(now)
        // log.Println(iteration_duration)
        if iteration_duration<MIN_UPDATE_PERIOD{
            time.Sleep(MIN_UPDATE_PERIOD-iteration_duration)
        }

        now=time.Now()
    }
}