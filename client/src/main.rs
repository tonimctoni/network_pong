// #![feature(box_syntax)]
extern crate sdl2;

// NETWORK IMPORTS
use std::net::{TcpStream, UdpSocket, IpAddr, SocketAddr};
use std::io;
use std::io::prelude::*;
use std::time::Duration;

// GAME IMPORTS
use sdl2::event::Event;
use sdl2::keyboard::Keycode;
use sdl2::pixels::Color;
use sdl2::rect::Rect;

// // THREAD IMPORTS
// use std::thread;
// use std::sync::{Arc, Mutex};
// use std::sync::atomic::{AtomicBool, Ordering};

// NETWORK CONSTANTS
const FIND_SERVER_PROTOCOL_LOCAL_ADDRESS: &str = "localhost:1234";
const FIND_SERVER_PROTOCOL_BROADCAST_ADDRESS: &str = "255.255.255.255:1234";
const LOOKING_FOR_SERVER_MESSAGE: &str = "cB9LFS";
const LOOKING_FOR_SERVER_RESPONSE: &str = "cB9IAS";
const CONNECT_TO_SERVER_MESSAGE: &str = "cB9CTS";
const DEFAULT_ADDRESS: &str = "0.0.0.0:0";
const CONNECT_PORT: u16 = 1235;
const FIND_SERVER_TIMEOUT_SECS: u64 = 1;

// GAME CONSTANTS
const PLAYER_WIDTH: u32 = 15;
const PLAYER_HEIGHT: u32 = 80;
const PLAYER_PADDING: i32 = 15;
const BALL_SIZE: u32 = 15;
const WINDOW_WIDTH: u32 = 800;
const WINDOW_HEIGHT: u32 = 600;
const WINDOW_TITLE: &str = "Network Pong";



fn get_server_ip(sendto_address: &str) -> io::Result<IpAddr>{
    let local_addr={
        let socket=UdpSocket::bind(DEFAULT_ADDRESS)?;
        socket.set_broadcast(true)?;

        let n=socket.send_to(LOOKING_FOR_SERVER_MESSAGE.as_bytes(), sendto_address)?;
        println!("[get_server_ip] Data sent to {}[{}]: {}", sendto_address, n, LOOKING_FOR_SERVER_MESSAGE);

        socket.local_addr()?
    };

    {
        let mut buffer=[0;100];
        let socket=UdpSocket::bind(local_addr)?;
        socket.set_read_timeout(Some(Duration::new(FIND_SERVER_TIMEOUT_SECS, 0)))?;

        let (n,peer_addr)=socket.recv_from(&mut buffer)?;
        if n==LOOKING_FOR_SERVER_RESPONSE.len(){
            let message=std::str::from_utf8(&mut buffer[..n]).unwrap_or(">ERROR");
            if message==LOOKING_FOR_SERVER_RESPONSE{
                Ok(peer_addr.ip())
            } else {
                Err(io::Error::new(io::ErrorKind::Other, "[get_server_ip] Wrong response (though it has the correct size)"))
            }
        } else {
            Err(io::Error::new(io::ErrorKind::Other, "[get_server_ip] Wrong response size"))
        }
    }
}

fn uint16_from_slice(slice: &[u8]) -> u16{
    return ((slice[1] as u16)<<8) | ((slice[0] as u16)<<0)
}

fn uint16_to_slice(n: u16, slice: &mut[u8]){
    slice[0]=((n>>0)&0xFF) as u8;
    slice[1]=((n>>8)&0xFF) as u8;
}

struct EventstateSender {
    ip: IpAddr,
    port: u16,
    socket: UdpSocket,
    eventstate: u16
}

impl EventstateSender{
    fn new(ip: IpAddr, port: u16) -> io::Result<EventstateSender>{
        Ok(EventstateSender{ip:ip, port:port, socket:UdpSocket::bind(DEFAULT_ADDRESS)?, eventstate:0})
    }

    fn send_eventstate(&self, iteration: u16) -> io::Result<()>{
        let mut buffer=[0;4];
        uint16_to_slice(iteration, &mut buffer[0..2]);
        uint16_to_slice(self.eventstate, &mut buffer[2..4]);
        self.socket.send_to(&buffer, SocketAddr::new(self.ip, self.port)).map(|_| ())
    }

    fn add_eventflag(&mut self, flag: u16){
        self.eventstate|=flag;
    }

    fn remove_eventflag(&mut self, flag: u16){
        self.eventstate&=0xFFFF^flag;
    }
}

struct GameState(i16, i16, i16, i16);

struct GamestateReceiver {
    socket: UdpSocket,
    received_messages: Vec<(u16, GameState)>
}

fn get_server_ports(server_ip: IpAddr) -> io::Result<(u16, u16)>{
    let mut stream = TcpStream::connect(SocketAddr::new(server_ip, CONNECT_PORT))?;

    stream.write(CONNECT_TO_SERVER_MESSAGE.as_bytes())?;
    let mut buffer=[0;100];
    let n=stream.read(&mut buffer)?;

    if n!=4{
        Err(io::Error::new(io::ErrorKind::Other, "[get_server_ports] Wrong response size"))
    } else {
        Ok((uint16_from_slice(&buffer[0..2]), uint16_from_slice(&buffer[2..4])))
    }
}

// struct SendEventMessagesStream {
//     stream: TcpStream
// }

// impl SendEventMessagesStream {
//     fn send_u16(&mut self, keycode: u16) -> io::Result<usize> {
//         let to_send=[((keycode>>0)&0xff) as u8, ((keycode>>8)&0xff) as u8];
//         self.stream.write(&to_send)
//     }
// }

// struct GetGameStateSocket {
//     socket: UdpSocket
// }

// // player1_pos, player2_pos, ball_pos_x, ball_pos_y
// #[derive(Copy, Clone)]
// struct GameState(i16, i16, i16, i16);
// impl GetGameStateSocket {
//     fn get_game_state(&self) -> io::Result<GameState>{
//         let mut buffer=[0;100];
//         let (n,_)=self.socket.recv_from(&mut buffer)?;

//         if n!=8{
//             Err(io::Error::new(io::ErrorKind::Other, "[get_game_state] Wrong response size"))
//         } else {
//             Ok(GameState(
//                 ((buffer[1] as i16)<<8) | ((buffer[0] as i16)<<0), 
//                 ((buffer[3] as i16)<<8) | ((buffer[2] as i16)<<0), 
//                 ((buffer[5] as i16)<<8) | ((buffer[4] as i16)<<0), 
//                 ((buffer[7] as i16)<<8) | ((buffer[6] as i16)<<0)
//                 ))
//         }
//     }
// }

// fn connect_to_server(server_ip: IpAddr) -> io::Result<(SendEventMessagesStream, GetGameStateSocket)>{
//     let stream = TcpStream::connect(SocketAddr::new(server_ip, CONNECT_PORT))?;
//     let socket = UdpSocket::bind(SocketAddr::new(server_ip, stream.local_addr()?.port()))?;
//     println!("[connect_to_server] Connection established: TCP({}->{}), UDP({}<-{})", stream.local_addr()?, stream.peer_addr()?, socket.local_addr()?, SocketAddr::new(server_ip, 0));

//     return Ok((SendEventMessagesStream{stream:stream}, GetGameStateSocket{socket:socket}));
// }


// TODO: REMOVE MAGIC NUMBERS
// TODO: Will probably need a queue for incomming messages like on the server
// TODO: Set timeout for get_game_state and atomic *finish* flag
fn main() {
    let server_ip=get_server_ip(FIND_SERVER_PROTOCOL_LOCAL_ADDRESS)
    .or_else( |_| get_server_ip(FIND_SERVER_PROTOCOL_BROADCAST_ADDRESS))
    .expect("Could not find server's ip");

    // let (mut stream, socket)=connect_to_server(server_ip).expect("Could not connect to server");

    let sdl_context = sdl2::init().unwrap();
    let video_subsystem = sdl_context.video().unwrap();
    let mut event_pump = sdl_context.event_pump().unwrap();

    let window = video_subsystem
    .window(WINDOW_TITLE, WINDOW_WIDTH, WINDOW_HEIGHT)
    .position_centered()
    // .opengl()
    .build()
    .unwrap();

    let mut canvas = window.into_canvas()
    // .target_texture()
    .present_vsync()
    .build()
    .unwrap();

    let mut p1_rect=Rect::new(PLAYER_PADDING,0,PLAYER_WIDTH,PLAYER_HEIGHT);
    let mut p2_rect=Rect::new((WINDOW_WIDTH-PLAYER_WIDTH) as i32 - PLAYER_PADDING,0,PLAYER_WIDTH,PLAYER_HEIGHT);
    let mut ball_rect=Rect::new((WINDOW_WIDTH/2-BALL_SIZE/2) as i32,(WINDOW_HEIGHT/2-BALL_SIZE/2) as i32,BALL_SIZE,BALL_SIZE);

    // let game_state=Arc::new(Mutex::new(GameState(0,0,0,0)));
    // let threads_gamestate=game_state.clone();
    // thread::spawn(move ||{
    //     loop{
    //         match socket.get_game_state() {
    //             Err(e) => println!("Error (socket.get_game_state): {}", e),
    //             Ok(new_game_state) => {
    //                 if let Ok(mut game_state) = threads_gamestate.lock(){
    //                     *game_state=new_game_state;
    //                 }
    //             }
    //         }
    //     }
    // });

    'running: loop{
        for event in event_pump.poll_iter() {
            match event {
                Event::Quit {..} | Event::KeyDown { keycode: Some(Keycode::Escape), .. } => {
                    // stream.send_u16(1).map(|_| ()).unwrap_or_else(|e| println!("Error (stream.send_u16): {}", e));
                    break 'running;
                }
                // Event::KeyDown { keycode: Some(Keycode::Up), .. } => stream.send_u16(2).map(|_| ()).unwrap_or_else(|e| println!("Error (stream.send_u16): {}", e)),
                // Event::KeyUp { keycode: Some(Keycode::Up), .. } => stream.send_u16(3).map(|_| ()).unwrap_or_else(|e| println!("Error (stream.send_u16): {}", e)),
                // Event::KeyDown { keycode: Some(Keycode::Down), .. } => stream.send_u16(4).map(|_| ()).unwrap_or_else(|e| println!("Error (stream.send_u16): {}", e)),
                // Event::KeyUp { keycode: Some(Keycode::Down), .. } => stream.send_u16(5).map(|_| ()).unwrap_or_else(|e| println!("Error (stream.send_u16): {}", e)),
                _ => (),
            }
        }

        // if let Ok(game_state)=game_state.try_lock(){
        //     p1_rect.set_y(game_state.0 as i32);
        //     p2_rect.set_y(game_state.1 as i32);
        //     ball_rect.set_x(game_state.2 as i32);
        //     ball_rect.set_y(game_state.3 as i32);
        // } else {
        //     continue;
        // }

        // match socket.get_game_state() {
        //     Err(e) => println!("Error (socket.get_game_state): {}", e),
        //     Ok(game_state) => {
        //         p1_rect.set_y(game_state.0 as i32);
        //         p2_rect.set_y(game_state.1 as i32);
        //         ball_rect.set_x(game_state.2 as i32);
        //         ball_rect.set_y(game_state.3 as i32);
        //     }
        // }



        canvas.set_draw_color(Color::RGB(0, 0, 0));
        canvas.clear();
        canvas.set_draw_color(Color::RGB(0, 200, 0));
        canvas.fill_rect(p1_rect).unwrap_or_else(|s| println!("Error (fill_rect): {}", s));
        canvas.fill_rect(p2_rect).unwrap_or_else(|s| println!("Error (fill_rect): {}", s));
        canvas.set_draw_color(Color::RGB(25, 255, 25));
        canvas.fill_rect(ball_rect).unwrap_or_else(|s| println!("Error (fill_rect): {}", s));
        canvas.present();
    }
}
