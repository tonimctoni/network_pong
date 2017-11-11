package main;



type Keystate struct{
    up bool
    down bool
}

type Player struct{
    keystate Keystate
    pos int16
}

func (p *Player) update_keystate(eventstate uint32){
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