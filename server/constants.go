package main;
import "time"



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



// type MessageBuffer struct{
//     pos int64
//     buffer [message_buffer_size]uint16
// }

// type MessageSender struct{
//     message_buffer *MessageBuffer
//     mb_sender chan *MessageBuffer
// }

// type MessageReceiver struct{
//     message_buffer *MessageBuffer
//     mb_receiver chan *MessageBuffer
//     pos int64
// }

// func create_message_transmission_pair() (MessageSender, MessageReceiver){
//     message_buffer_ptr:=&MessageBuffer{}
//     mbc:=make(chan *MessageBuffer, 2)

//     return MessageSender{message_buffer_ptr, mbc}, MessageReceiver{message_buffer_ptr, mbc, 0}
// }

// func (m *MessageSender) send(message uint16){
//     pos:=atomic.LoadInt64(&m.message_buffer.pos)

//     m.message_buffer.buffer[pos]=message
//     atomic.AddInt64(&m.message_buffer.pos, 1)
//     pos+=1

//     if pos>message_buffer_size{
//         panic("pos>message_buffer_size")
//     }

//     if pos==message_buffer_size{
//         new_message_buffer_ptr:=&MessageBuffer{}
//         m.message_buffer=new_message_buffer_ptr
//         m.mb_sender<-new_message_buffer_ptr
//     }
// }

// func (m *MessageReceiver) receive() []uint16{
//     if m.pos>message_buffer_size{
//         panic("m.pos>message_buffer_size")
//     }

//     if m.pos==message_buffer_size{
//         m.message_buffer=<-m.mb_receiver
//         m.pos=0
//     }

//     pos:=atomic.LoadInt64(&m.message_buffer.pos)
//     to_return:=m.message_buffer.buffer[m.pos:pos]
//     m.pos=pos
//     return to_return
// }
