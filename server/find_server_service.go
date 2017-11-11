package main;
import "net"
import "log"
import "time"



func find_server_service() {
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