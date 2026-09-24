export const DEFAULT_FILE_NAME = 'sum.s';

export const DEFAULT_PROGRAM = `; WinMIPS64 - sums an array, does some FP math and prints the sum
        .data
array:  .word 3, 5, 7, 11
len:    .word 4
sum:    .word 0
x:      .double 1.5
y:      .double 2.25
CONTROL: .word32 0x10000
DATA:    .word32 0x10008

        .text
main:   ld    r1, len(r0)
        daddi r2, r0, array
        daddu r3, r0, r0
loop:   ld    r4, 0(r2)
        daddu r3, r3, r4
        daddi r2, r2, 8
        daddi r1, r1, -1
        bnez  r1, loop
        sd    r3, sum(r0)
        l.d   f1, x(r0)
        l.d   f2, y(r0)
        mul.d f3, f1, f2
        add.d f4, f3, f1
        lwu   r5, CONTROL(r0)
        lwu   r6, DATA(r0)
        sd    r3, 0(r6)
        daddi r7, r0, 1
        sd    r7, 0(r5)
        halt
`;
