package main;



type Ball struct{
    pos_x int16
    pos_y int16
    speed_x int16
    speed_y int16
}

func (b *Ball) move_y(speed_y int16) bool{
    b.pos_y+=speed_y
    if b.pos_y<0{
        b.pos_y=0
        b.speed_y=-b.speed_y
        return true
    }
    if b.pos_y+BALL_SIZE>=WINDOW_HEIGHT{
        b.pos_y=WINDOW_HEIGHT-BALL_SIZE-1
        b.speed_y=-b.speed_y
        return true
    }
    return false
}

func (b *Ball) move_x(speed_x int16) bool{
    b.pos_x+=speed_x
    if b.pos_x<0{
        b.pos_x=0
        b.speed_x=-b.speed_x
        return true
    }
    if b.pos_x+BALL_SIZE>=WINDOW_WIDTH{
        b.pos_x=WINDOW_WIDTH-BALL_SIZE-1
        b.speed_x=-b.speed_x
        return true
    }
    return false
}

func (b *Ball) move(p1pos, p2pos int16) int{
    if b.pos_x+b.speed_x<PLAYER_PADDING+PLAYER_WIDTH && b.pos_x>PLAYER_PADDING+PLAYER_WIDTH{
        portion_moved:=float64(PLAYER_PADDING+PLAYER_WIDTH-b.pos_x)/float64(b.speed_x)
        this_speed_x:=int16(portion_moved*float64(b.speed_x))
        this_speed_y:=int16(portion_moved*float64(b.speed_y))

        moved_portion_y:=b.pos_y+this_speed_y
        if p1pos<=moved_portion_y+BALL_SIZE && p1pos+PLAYER_HEIGHT>=moved_portion_y{
            b.pos_x+=this_speed_x
            b.speed_x=-b.speed_x
            b.move_y(b.speed_y)
            return 0
        } else {
            if b.move_x(b.speed_x){
                b.pos_x=WINDOW_WIDTH/2-BALL_SIZE/2
                b.pos_y=WINDOW_HEIGHT/2-BALL_SIZE/2
                b.speed_x=BALL_SPEED
                b.speed_y=-BALL_SPEED
                if b.speed_x<0{
                    return 2
                } else if b.speed_x>0{
                    return 1
                } else {
                    return 0
                }
            }
            b.move_y(b.speed_y)
        }
    } else if b.pos_x+b.speed_x>WINDOW_WIDTH-(PLAYER_PADDING+PLAYER_WIDTH) && b.pos_x<WINDOW_WIDTH-(PLAYER_PADDING+PLAYER_WIDTH){
        b.pos_x-=b.speed_x
        b.speed_x=-b.speed_x
        b.move_y(b.speed_y)
        return 0
    } else {
        if b.move_x(b.speed_x){
            b.pos_x=WINDOW_WIDTH/2-BALL_SIZE/2
            b.pos_y=WINDOW_HEIGHT/2-BALL_SIZE/2
            b.speed_x=BALL_SPEED
            b.speed_y=-BALL_SPEED
            if b.speed_x<0{
                return 2
            } else if b.speed_x>0{
                return 1
            } else {
                return 0
            }
        }
        b.move_y(b.speed_y)
        return 0
    }
    return 0
}
