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
use std::thread;
use std::sync::{Arc, Mutex};
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


fn uint16_from_slice(slice: &[u8]) -> u16{
    ((slice[1] as u16)<<8) | ((slice[0] as u16)<<0)
}

fn int16_from_slice(slice: &[u8]) -> i16{
    ((slice[1] as i16)<<8) | ((slice[0] as i16)<<0)
}

fn uint16_to_slice(n: u16, slice: &mut[u8]){
    slice[0]=((n>>0)&0xFF) as u8;
    slice[1]=((n>>8)&0xFF) as u8;
}

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
}

impl GamestateReceiver {
    fn new(ip: IpAddr, port: u16) -> io::Result<GamestateReceiver>{
        Ok(GamestateReceiver{socket:UdpSocket::bind(SocketAddr::new(ip, port))?})
    }

    fn get_game_state(&self) -> io::Result<(u16, GameState)>{
        let mut buffer=[0;100];
        let (n,_)=self.socket.recv_from(&mut buffer)?;

        if n!=10{
            Err(io::Error::new(io::ErrorKind::Other, "[get_game_state] Wrong response size"))
        } else {
            Ok((uint16_from_slice(&buffer[0..2]),
                GameState(int16_from_slice(&buffer[2..4]),
                    int16_from_slice(&buffer[4..6]),
                    int16_from_slice(&buffer[6..8]),
                    int16_from_slice(&buffer[8..10])
                    )))
        }
    }
}

fn connect_to_server(server_ip: IpAddr) -> io::Result<(u8, GamestateReceiver, EventstateSender)>{
    let mut stream = TcpStream::connect(SocketAddr::new(server_ip, CONNECT_PORT))?;

    stream.write(CONNECT_TO_SERVER_MESSAGE.as_bytes())?;
    let mut buffer=[0;100];
    let n=stream.read(&mut buffer)?;

    if n!=5{
        Err(io::Error::new(io::ErrorKind::Other, "[connect_to_server] Wrong response size"))
    } else {
        // Ok((uint16_from_slice(&buffer[0..2]), uint16_from_slice(&buffer[2..4])))
        Ok((buffer[0],
            GamestateReceiver::new(server_ip, uint16_from_slice(&buffer[1..3]))?,
            EventstateSender::new(server_ip, uint16_from_slice(&buffer[3..5]))?)
        )
    }
}

// TODO: REMOVE MAGIC NUMBERS
// TODO: Will probably need a queue for incomming messages like on the server
// TODO: Set timeout for get_game_state and atomic *finish* flag
fn main() {
    let server_ip=get_server_ip(FIND_SERVER_PROTOCOL_LOCAL_ADDRESS)
    .or_else( |_| get_server_ip(FIND_SERVER_PROTOCOL_BROADCAST_ADDRESS))
    .expect("Could not find server's ip");

    let (_, gamestate_receiver, mut eventstate_sender)=connect_to_server(server_ip).expect("Could not connect to server");

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

    let received_gamestate=Arc::new(Mutex::new((0,GameState(0,0,0,0))));
    let received_gamestate_c=received_gamestate.clone();
    thread::spawn(move ||{
        loop{
            match gamestate_receiver.get_game_state(){
                Err(e) => println!("Error (gamestate_receiver.get_game_state): {}", e),
                Ok(new_received_gamestate)=>{
                    match received_gamestate_c.lock(){
                        Err(e) => println!("Error (received_gamestate_c.lock): {}", e),
                        Ok(mut received_gamestate) => *received_gamestate=new_received_gamestate,
                    }
                }
            }
        }
    });

    'running: loop{
        for event in event_pump.poll_iter() {
            match event {
                Event::Quit {..} | Event::KeyDown { keycode: Some(Keycode::Escape), .. } => {
                    eventstate_sender.add_eventflag(1);
                    // eventstate_sender.send_eventstate(0).map(|_| ()).unwrap_or_else(|e| println!("Error (eventstate_sender.send_eventstate): {}", e));
                    break 'running;
                }
                Event::KeyDown { keycode: Some(Keycode::Up), .. } => eventstate_sender.add_eventflag(2),
                Event::KeyUp { keycode: Some(Keycode::Up), .. } => eventstate_sender.remove_eventflag(2),
                Event::KeyDown { keycode: Some(Keycode::Down), .. } => eventstate_sender.add_eventflag(4),
                Event::KeyUp { keycode: Some(Keycode::Down), .. } => eventstate_sender.remove_eventflag(4),
                _ => (),
            }
        }

        eventstate_sender.send_eventstate(0).map(|_| ()).unwrap_or_else(|e| println!("Error (eventstate_sender.send_eventstate): {}", e));

        if let Ok(received_gamestate)=received_gamestate.try_lock(){
            p1_rect.set_y((received_gamestate.1).0 as i32);
            p2_rect.set_y((received_gamestate.1).1 as i32);
            ball_rect.set_x((received_gamestate.1).2 as i32);
            ball_rect.set_y((received_gamestate.1).3 as i32);
        } else {
            //make it last a frame here (or not, measure)
            continue;
        }

        //make it last a frame here also

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
