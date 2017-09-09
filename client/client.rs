// shiro: Same as with `try!`: Preferrably your own Error type with an `impl From<OtherErrorType>`. There are a few helper libraries out there to simplify this, but usually one defines an Enum like enum `Error { Io(io::Error), Encoding(Utf8Error), ... }` for t
#![feature(question_mark)]
use std::net::{TcpStream, UdpSocket, IpAddr, SocketAddr};
// use std::net::;
use std::io;
use std::io::prelude::*;
use std::time::Duration;

static FIND_SERVER_PROTOCOL_LOCAL_ADDRESS: &'static str = "localhost:1234";
static FIND_SERVER_PROTOCOL_BROADCAST_ADDRESS: &'static str = "255.255.255.255:1234";
static LOOKING_FOR_SERVER_MESSAGE: &'static str = "cB9LFS";
static LOOKING_FOR_SERVER_RESPONSE: &'static str = "cB9IAS";
static DEFAULT_ADDRESS: &'static str = "0.0.0.0:0";
const CONNECT_PORT: u16 = 1235;
const FIND_SERVER_TIMEOUT_SECS: u64 = 1;


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

struct SendKeyboardMessagesStream {
    stream: TcpStream
}

impl SendKeyboardMessagesStream {
    fn send_u16(&mut self, keycode: u16) -> io::Result<usize> {
        let to_send=[((keycode>>8)&0xff) as u8, ((keycode>>0)&0xff) as u8];
        self.stream.write(&to_send)
    }
}

struct GetGameStateSocket {
    socket: UdpSocket
}

fn connect_to_server(server_ip: IpAddr) -> io::Result<(SendKeyboardMessagesStream, GetGameStateSocket)>{
    let stream = TcpStream::connect(SocketAddr::new(server_ip, CONNECT_PORT))?;
    let socket = UdpSocket::bind(SocketAddr::new(server_ip, stream.local_addr()?.port()))?;
    println!("[connect_to_server] Connection established: TCP({}->{}), UDP({}<-{})", stream.local_addr()?, stream.peer_addr()?, socket.local_addr()?, SocketAddr::new(server_ip, 0));

    return Ok((SendKeyboardMessagesStream{stream:stream}, GetGameStateSocket{socket:socket}));
}


fn main() {
    let server_ip=get_server_ip(FIND_SERVER_PROTOCOL_LOCAL_ADDRESS)
    .or_else( |_| {get_server_ip(FIND_SERVER_PROTOCOL_BROADCAST_ADDRESS)})
    .expect("Could not find server's ip");

    println!("{:?}", server_ip);
    let (stream, socket)=connect_to_server(server_ip).expect("Could not connect to server");

}
// shiro: Same as with `try!`: Preferrably your own Error type with an `impl From<OtherErrorType>`. There are a few helper libraries out there to simplify this, but usually one defines an Enum like enum `Error { Io(io::Error), Encoding(Utf8Error), ... }` for t