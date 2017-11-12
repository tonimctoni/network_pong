package main;
import "net"
import "log"
import "strings"
import "errors"
import "sync"



func get_ip_and_port_from_address(address string) (string, string, error) {
    i:=strings.LastIndex(address, ":")
    if i<0 {
        return "", "", errors.New("Weird address")
    }
    return address[:i], address[i+1:], nil
}

type PlayerConnection struct{
    gamestate_port uint16
    eventstate_port uint16
    ip_address string
}

func accept_incomming_connections(pcc chan PlayerConnection){
    listener,err:=net.Listen("tcp4", connect_address)
    if err!=nil{
        log.Fatalln("[accept_incomming_connections] Error (net.Listen):", err)
    }
    defer listener.Close()
    log.Println("[accept_incomming_connections] Listening on:", listener.Addr())

    port_counter:=uint16(port_counter_start)
    player_num_mutex:=&sync.Mutex{}
    player_num:=byte(1)
    for{
        connection, err:=listener.Accept()
        if err != nil {
            log.Println("[accept_incomming_connections] Error (listener.Accept):", err)
            continue
        }

        port_counter+=2
        if port_counter<port_counter_start{
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

            player_num_mutex.Lock()
            defer player_num_mutex.Unlock()
            buffer[0]=player_num
            player_num=(player_num%2)+1

            uint16_to_slice(gamestate_port, buffer[1:3])
            uint16_to_slice(eventstate_port, buffer[3:5])
            _,err=connection.Write(buffer[:5])
            if err!=nil{
                log.Println("[accept_incomming_connections] Error (connection.Write):", err)
                player_num=buffer[0]
                return
            }
            log.Println("[accept_incomming_connections] Connection established for:", connection.RemoteAddr())

            pcc<-PlayerConnection{gamestate_port, eventstate_port, ip_address}
        }(connection, port_counter, port_counter+1)
    }
}