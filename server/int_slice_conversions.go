package main;



func uint16_from_slice(slice []byte) uint16{
    return (uint16(slice[1])<<8) | (uint16(slice[0])<<0)
}

func uint32_from_slice(slice []byte) uint32{
    return (uint32(slice[3])<<24) | (uint32(slice[2])<<16) | (uint32(slice[1])<<8) | (uint32(slice[0])<<0)
}

func uint16_to_slice(n uint16, slice []byte){
    slice[0]=byte((n>>0)&0xFF)
    slice[1]=byte((n>>8)&0xFF)
}

func int16_to_slice(n int16, slice []byte){
    slice[0]=byte((n>>0)&0xFF)
    slice[1]=byte((n>>8)&0xFF)
}