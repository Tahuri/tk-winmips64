; Oracle self-test for the memory-mapped I/O (CONTROL / DATA) and keyboard input.
        .data
CONTROL: .word32 0x10000
DATA:    .word32 0x10008
msg:     .asciiz "Enter:"
pix:     .word 0x00000A0500FF0000   ; x=10 y=5 colour 0x00ff0000
        .text
        lwu r8,CONTROL(r0)
        lwu r9,DATA(r0)
        daddi r10,r0,msg
        sd r10,(r9)
        daddi r11,r0,4
        sd r11,(r8)          ; print string
        daddi r11,r0,8
        sd r11,(r8)          ; read integer
        ld r12,(r9)
        sd r12,(r9)
        daddi r11,r0,2
        sd r11,(r8)          ; print signed integer
        daddi r11,r0,8
        sd r11,(r8)          ; read double
        l.d f1,(r9)
        s.d f1,(r9)
        daddi r11,r0,3
        sd r11,(r8)          ; print double
        daddi r11,r0,9
        sd r11,(r8)          ; read character
        lbu r12,(r9)
        sd r12,(r9)
        daddi r11,r0,1
        sd r11,(r8)          ; print unsigned
        ld r13,pix(r0)
        sd r13,(r9)
        daddi r11,r0,5
        sd r11,(r8)          ; plot pixel
        halt
