main:
    lr $61 %0 # a
    lr $7b %1 # z + 1
    
    loop:
        p %0
        addi %0 $1 %0
        bne %0 %1 loop

    nop
    lr $0a %0 # \n
    p %0

    lr $40 %0
    call func
    exit

func:
    lr $0 %1
func_loop:
    addi %1 $2 %1
    blt %1 %0 func_loop
    jr %3
