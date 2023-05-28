package cpu

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/rtcall/hypo/asm"
)

type Cpu struct {
	reg   [8]uint32
	mem   [8192]byte
	pc    uint32
	flags uint32
	err   error
	buf   *bytes.Reader
}

func New(buf []byte) (c Cpu, err error) {
	c.buf = bytes.NewReader(buf)

	hdr := make([]byte, len(asm.Hdr))
	if _, err = c.buf.Read(hdr); err != nil {
		return c, errors.New("could not read header")
	} else if !bytes.Equal(hdr, asm.Hdr) {
		return c, errors.New("bad header")
	}

	return c, nil
}

func (c *Cpu) read(ins any) {
	if err := binary.Read(c.buf, binary.LittleEndian, ins); err != nil {
		c.err = errors.New("bad read")
	}
}

func (c *Cpu) checkReg(r byte) error {
	if r > byte(len(c.reg)) {
		c.err = fmt.Errorf("invalid register %02x", r)
		c.flags |= 1
	}

	return c.err
}

func (c *Cpu) readReg(r byte) uint32 {
	if c.checkReg(r) == nil {
		return c.reg[r]
	}

	return 0
}

func (c *Cpu) writeReg(r byte, i uint32) {
	if c.checkReg(r) == nil {
		c.reg[r] = i
	}
}

func (c *Cpu) readImm(addr uint32) (uint32, error) {
	if addr > uint32(len(c.mem)) {
		return 0, fmt.Errorf("illegal read %08x", addr)
	}

	i := c.mem[addr:4]
	return uint32(i[3])<<24 | uint32(i[2])<<16 | uint32(i[1])<<8 | uint32(i[0]), nil
}

func (c *Cpu) writeImm(addr, imm uint32) error {
	if addr > uint32(len(c.mem)) {
		return fmt.Errorf("illegal write %08x (at %08x)", imm, addr)
	}

	c.mem[addr] = byte(imm)
	c.mem[addr+1] = byte(imm >> 8)
	c.mem[addr+2] = byte(imm >> 16)
	c.mem[addr+3] = byte(imm >> 24)
	return nil
}

func (c *Cpu) jump(pc uint32) error {
	if _, err := c.buf.Seek(int64(pc+uint32(len(asm.Hdr))), io.SeekStart); err != nil {
		return err
	}

	c.pc = pc
	return nil
}

var ops = map[byte]func(*Cpu) int{
	asm.OpNop: func(*Cpu) int {
		return 0
	},
	asm.OpLd: func(c *Cpu) int {
		var ins struct {
			R1, R2 byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		i, err := c.readImm(c.readReg(ins.R2))
		c.err = err

		if err != nil {
			c.writeReg(ins.R1, i)
		}

		return 2
	},
	asm.OpLr: func(c *Cpu) int {
		var ins struct {
			I uint32
			R byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		c.writeReg(ins.R, ins.I)
		return 5
	},
	asm.OpSt: func(c *Cpu) int {
		var ins struct {
			R1, R2 byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		c.err = c.writeImm(c.readReg(ins.R1), c.readReg(ins.R2))
		return 2
	},
	asm.OpAdd: func(c *Cpu) int {
		var ins struct {
			R1, R2, R3 byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		c.writeReg(ins.R3, c.readReg(ins.R1)+c.readReg(ins.R2))
		return 3
	},
	asm.OpSub: func(c *Cpu) int {
		var ins struct {
			R1, R2, R3 byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		c.writeReg(ins.R3, c.readReg(ins.R1)-c.readReg(ins.R2))
		return 3
	},
	asm.OpAddi: func(c *Cpu) int {
		var ins struct {
			R1 byte
			I  uint32
			R2 byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		c.writeReg(ins.R2, c.readReg(ins.R1)+ins.I)
		return 6
	},
	asm.OpSubi: func(c *Cpu) int {
		var ins struct {
			R1 byte
			I  uint32
			R2 byte
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		c.writeReg(ins.R2, c.readReg(ins.R1)-ins.I)
		return 6
	},
	asm.OpP: func(c *Cpu) int {
		var R byte

		if c.read(&R); c.err != nil {
			return 0
		}

		fmt.Print(string(rune(c.reg[R])))
		return 1
	},
	asm.OpBeq: func(c *Cpu) int {
		var ins struct {
			R1, R2 byte
			I      uint32
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		if c.readReg(ins.R1) == c.readReg(ins.R2) {
			c.jump(ins.I)
			return 0
		}

		return 6
	},
	asm.OpBne: func(c *Cpu) int {
		var ins struct {
			R1, R2 byte
			I      uint32
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		if c.readReg(ins.R1) != c.readReg(ins.R2) {
			c.jump(ins.I)
			return 0
		}

		return 6
	},
	asm.OpBgt: func(c *Cpu) int {
		var ins struct {
			R1, R2 byte
			I      uint32
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		if c.readReg(ins.R1) > c.readReg(ins.R2) {
			c.jump(ins.I)
			return 0
		}

		return 6
	},
	asm.OpBlt: func(c *Cpu) int {
		var ins struct {
			R1, R2 byte
			I      uint32
		}

		if c.read(&ins); c.err != nil {
			return 0
		}

		if c.readReg(ins.R1) < c.readReg(ins.R2) {
			c.jump(ins.I)
			return 0
		}

		return 6
	},
	asm.OpJ: func(c *Cpu) int {
		var I uint32

		if c.read(&I); c.err != nil {
			return 0
		}

		c.jump(I)
		return 0
	},
	asm.OpJr: func(c *Cpu) int {
		var R byte

		if c.read(&R); c.err != nil {
			return 0
		}

		c.jump(c.readReg(R))
		return 0
	},
	asm.OpCall: func(c *Cpu) int {
		var I uint32

		if c.read(&I); c.err != nil {
			return 0
		}

		c.writeReg(3, c.pc+4)
		c.jump(I)
		return 0
	},
	asm.OpExit: func(c *Cpu) int {
		c.flags |= 1
		return 0
	},
}

func (c *Cpu) State() bool {
	return c.flags != 1
}

func (c *Cpu) Step() error {
	var op byte

	if c.read(&op); c.err != nil {
		return c.err
	}

	c.pc++

	f, ok := ops[op]
	if !ok {
		c.pc--
		return fmt.Errorf("invalid opcode: %02x", op)
	}

	pc := f(c)
	c.pc += uint32(pc)
	return c.err
}

func (c *Cpu) WriteTrace(w io.Writer) {
	fmt.Fprintln(w, "register trace:")
	for i, j := range c.reg {
		fmt.Fprintf(w, "%02x: %08x\n", i, j)
	}

	fmt.Fprintf(w, "pc: %08x\n", c.pc)
	fmt.Fprintln(w, "memory trace:")
	for i, j := range c.mem {
		if i > 0xff {
			break
		}
		if i > 0 && i%16 == 0 {
			fmt.Fprintln(w, "")
		}
		fmt.Fprintf(w, "%02x ", j)
	}

	fmt.Fprintln(w, "")
}
