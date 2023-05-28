package asm

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

const (
	Id = iota
	Label
	Reg
	Addr
	Eof
)

const ErrThreshold = 8

type Symbol struct {
	Type int
	Val  string
	Line int
}

type Instruction struct {
	Op     byte
	Params []int
}

type Reader struct {
	sym  []Symbol
	nsym int
}

type Writer struct {
	buf  bytes.Buffer
	pc   uint32
	lab  map[string]uint32
	addr map[uint32]string
	f    io.Writer
}

var Hdr = []byte{0x48, 0x59, 0x50, 0x00}

var syms = map[byte]int{
	'%': Reg,
	'$': Addr,
}

var inst = map[string]Instruction{
	"nop":  {OpNop, []int{}},
	"ld":   {OpLd, []int{Reg, Reg}},
	"lr":   {OpLr, []int{Addr, Reg}},
	"st":   {OpSt, []int{Reg, Reg}},
	"add":  {OpAdd, []int{Reg, Reg, Reg}},
	"sub":  {OpSub, []int{Reg, Reg, Reg}},
	"addi": {OpAddi, []int{Reg, Addr, Reg}},
	"subi": {OpSubi, []int{Reg, Addr, Reg}},
	"p":    {OpP, []int{Reg}},
	"beq":  {OpBeq, []int{Reg, Reg, Addr}},
	"bne":  {OpBne, []int{Reg, Reg, Addr}},
	"bgt":  {OpBgt, []int{Reg, Reg, Addr}},
	"blt":  {OpBlt, []int{Reg, Reg, Addr}},
	"j":    {OpJ, []int{Addr}},
	"jr":   {OpJr, []int{Reg}},
	"call": {OpCall, []int{Addr}},
	"exit": {OpExit, []int{}},
}

var lc int

func NewReader(s []Symbol) *Reader {
	r := new(Reader)
	r.sym = s
	return r
}

func NewWriter(w io.Writer) *Writer {
	r := new(Writer)
	r.lab = make(map[string]uint32)
	r.addr = make(map[uint32]string)
	r.f = w
	return r
}

func (s *Reader) Read() (Symbol, error) {
	if s.nsym == len(s.sym) {
		return Symbol{}, errors.New("bad argument count")
	}

	sym := s.sym[s.nsym]
	s.nsym++
	return sym, nil
}

func (s *Reader) Expect(t int) (Symbol, error) {
	sym, err := s.Read()

	if err != nil {
		return sym, err
	}

	switch t {
	case Id:
		if sym.Type != t && sym.Type != Label {
			return sym, fmt.Errorf("expected identifier got '%s'", sym.Val)
		}
	default:
		if sym.Type != t && sym.Type != Id {
			tval := "register"
			if t == Addr {
				tval = "immediate"
			}

			return sym, fmt.Errorf("expected %s got '%s'", tval, sym.Val)
		}
	}

	return sym, nil
}

func (w *Writer) WriteAddr(addr uint32) {
	binary.Write(&w.buf, binary.LittleEndian, addr)
	w.pc += 4
}

func (w *Writer) WriteSymbol(sym Symbol) error {
	switch sym.Type {
	case Id:
		if f, ok := inst[sym.Val]; ok {
			w.buf.WriteByte(f.Op)
			w.pc++
		} else {
			w.addr[w.pc] = sym.Val
			w.WriteAddr(0)
		}
	case Label:
		_, ok := w.lab[sym.Val]

		if ok {
			return fmt.Errorf("redefining label '%s'", sym.Val)
		}

		w.lab[sym.Val] = w.pc
	case Reg:
		r, err := strconv.Atoi(sym.Val)

		if err != nil {
			return fmt.Errorf("bad register '%s'", sym.Val)
		}

		w.buf.WriteByte(byte(r))
		w.pc++
	case Addr:
		addr, err := strconv.ParseInt(sym.Val, 16, 32)

		if err != nil {
			return fmt.Errorf("bad address '%s'", sym.Val)
		}

		w.WriteAddr(uint32(addr))
	}

	return nil
}

func (w *Writer) Write() (int, error) {
	b := w.buf.Bytes()

	for i, j := range w.addr {
		l, ok := w.lab[j]

		if !ok {
			return -1, fmt.Errorf("%s: no such label", j)
		}

		b[i] = byte(l)
		b[i+1] = byte(l >> 8)
		b[i+2] = byte(l >> 16)
		b[i+3] = byte(l >> 24)
	}

	if n, err := w.f.Write(Hdr); err != nil {
		return n, err
	}

	return w.f.Write(b)
}

func ReadToken(r *bufio.Reader) (string, error) {
	b := new(bytes.Buffer)

	for {
		c, err := r.ReadByte()

		if err != nil {
			return "", err
		}

		if c == '\n' {
			lc++
		}

		if unicode.IsSpace(rune(c)) {
			break
		}

		b.WriteByte(c)
	}

	return b.String(), nil
}

func Read(r *bufio.Reader) (sym Symbol, err error) {
	sym.Type = -1

	for {
		c, err := r.ReadByte()

		if err != nil {
			sym.Type = Eof
			break
		}

		switch c {
		case '\n':
			lc++
		case '#':
			r.ReadBytes('\n')
			lc++
			return sym, nil
		}

		if unicode.IsSpace(rune(c)) {
			continue
		}

		if !unicode.IsGraphic(rune(c)) {
			return sym, fmt.Errorf("invalid character '%02x'", c)
		}

		if t, ok := syms[c]; ok {
			s, err := ReadToken(r)

			if err != nil {
				sym.Type = Eof
			} else {
				sym = Symbol{t, s, lc}
			}

			break
		}

		if sym.Type == -1 && unicode.IsLetter(rune(c)) {
			r.UnreadByte()
			s, err := ReadToken(r)

			if err != nil {
				sym.Type = Eof
			} else {
				if s[len(s)-1] == ':' {
					sym = Symbol{Label, strings.TrimSuffix(s, ":"), lc}
					lc++
				} else {
					sym = Symbol{Id, s, lc}
				}
			}

			break
		}
	}

	return sym, nil
}

// Gen takes the code from r and writes a machine code representation
// to w. Any errors are outputted to e.
func Gen(r io.Reader, w io.Writer, e io.Writer) (sym []Symbol, err error) {
	b := bufio.NewReader(r)
	errc := 0

	werr := func(s Symbol, err error) {
		if errc <= ErrThreshold {
			fmt.Fprintf(e, "%d: %s\n", s.Line, err)
		}
		errc++
	}

	for {
		s, err := Read(b)

		if err != nil {
			werr(s, err)
		}

		if errc > ErrThreshold {
			return sym, errors.New("invalid file")
		}

		if s.Type != -1 {
			sym = append(sym, s)
		}

		if s.Type == Eof {
			break
		}
	}

	reader := NewReader(sym)
	writer := NewWriter(w)

	for {
		s, err := reader.Expect(Id)

		if s.Type == Eof {
			break
		}

		if err != nil {
			werr(s, err)
			continue
		}

		if s.Type != Label {
			f, ok := inst[s.Val]

			if !ok {
				werr(s, fmt.Errorf("bad instruction '%s'", s.Val))
				continue
			}

			writer.WriteSymbol(s)
			for _, t := range f.Params {
				s, err = reader.Expect(t)

				if err != nil {
					werr(s, err)
				} else if err = writer.WriteSymbol(s); err != nil {
					werr(s, err)
				}
			}

			continue
		}

		if err = writer.WriteSymbol(s); err != nil {
			werr(s, err)
		}
	}

	if errc > ErrThreshold {
		return sym, fmt.Errorf("%d errors (%d shown)", errc, ErrThreshold)
	} else if errc > 0 {
		return sym, fmt.Errorf("%d errors", errc)
	}

	if _, err := writer.Write(); err != nil {
		fmt.Fprintf(e, "%s\n", err)
	}

	return sym, nil
}
