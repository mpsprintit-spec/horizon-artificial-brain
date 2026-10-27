package teaching

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const magic uint32 = 0x314c4e48

var order = binary.LittleEndian

// HNL/1 uses typed integers, IEEE-754 binary64 and a CRC32. CRC detects
// accidental corruption; it is not authentication or encryption.
func Marshal(p Packet) ([]byte, error) {
	p = canonicalPacket(p)
	if err := p.Validate(); err != nil {
		return nil, err
	}
	b := new(bytes.Buffer)
	w := func(v any) { _ = binary.Write(b, order, v) }
	flag := func(v bool) uint8 {
		if v {
			return 1
		}
		return 0
	}
	w(magic)
	w(p.Version)
	w(uint8(p.Kind))
	w(p.ID)
	w(p.Source)
	w(p.Stream)
	w(p.At)
	w(p.Question.Domain)
	w(uint8(p.Question.Relation))
	w(p.Question.A)
	w(p.Question.B)
	if p.Version == ExtendedVersion {
		w(p.Question.C)
		w(p.Question.D)
	}
	w(uint16(len(p.Examples)))
	for _, e := range p.Examples {
		w(uint8(e.Role))
		w(e.Episode)
		w(e.ObservedAt)
		w(uint8(len(e.Samples)))
		for _, s := range e.Samples {
			w(s.Lag)
			w(flag(s.Present))
			w(s.Values)
		}
		w(flag(e.Outcome != nil))
		if e.Outcome != nil {
			w(e.Outcome.Value)
			w(e.Outcome.ObservedAt)
		}
	}
	w(flag(p.Reply != nil))
	if r := p.Reply; r != nil {
		w(r.Sequence)
		w(r.Status)
		w(r.Value)
		w(r.CheckRMSE)
		w(uint8(len(r.Units)))
		for _, u := range r.Units {
			w(u.Node)
			w(u.Code)
			w(u.Resolution)
			w(u.Contribution)
		}
		w(r.LearnedN)
		w(r.Cause)
		w(flag(r.Committed))
		w(r.BeforeError)
		w(r.AfterError)
		w(r.Grown)
		w(r.ResolutionBefore)
		w(r.ResolutionAfter)
		w(uint8(len(r.Trials)))
		for _, t := range r.Trials {
			w(t.Kind)
			w(t.Node)
			w(t.Error)
			w(t.Regressions)
			w(flag(t.Eligible))
		}
	}
	if b.Len()+4 > MaxPacketBytes {
		return nil, errors.New("HNL packet exceeds resource bound")
	}
	w(crc32.ChecksumIEEE(b.Bytes()))
	return b.Bytes(), nil
}

func Unmarshal(data []byte) (Packet, error) {
	var p Packet
	if len(data) < 4 || len(data) > MaxPacketBytes || crc32.ChecksumIEEE(data[:len(data)-4]) != order.Uint32(data[len(data)-4:]) {
		return p, errors.New("HNL length or checksum mismatch")
	}
	r := bytes.NewReader(data[:len(data)-4])
	var err error
	rd := func(v any) {
		if err == nil {
			err = binary.Read(r, order, v)
		}
	}
	byteValue := func() uint8 { var v uint8; rd(&v); return v }
	boolValue := func() bool {
		v := byteValue()
		if v > 1 && err == nil {
			err = errors.New("invalid HNL boolean")
		}
		return v == 1
	}
	var m uint32
	rd(&m)
	rd(&p.Version)
	p.Kind = Kind(byteValue())
	rd(&p.ID)
	rd(&p.Source)
	rd(&p.Stream)
	rd(&p.At)
	rd(&p.Question.Domain)
	p.Question.Relation = Relation(byteValue())
	rd(&p.Question.A)
	rd(&p.Question.B)
	if p.Version == ExtendedVersion {
		rd(&p.Question.C)
		rd(&p.Question.D)
	}
	var count uint16
	rd(&count)
	if count > 128 {
		return Packet{}, errors.New("HNL observation bound exceeded")
	}
	for i := 0; i < int(count) && err == nil; i++ {
		e := Example{Role: Role(byteValue())}
		rd(&e.Episode)
		rd(&e.ObservedAt)
		n := byteValue()
		if n > 64 {
			return Packet{}, errors.New("HNL lag bound exceeded")
		}
		for j := 0; j < int(n) && err == nil; j++ {
			s := Sample{Lag: byteValue(), Present: boolValue()}
			rd(&s.Values)
			e.Samples = append(e.Samples, s)
		}
		if boolValue() {
			e.Outcome = &Evidence{}
			rd(&e.Outcome.Value)
			rd(&e.Outcome.ObservedAt)
		}
		p.Examples = append(p.Examples, e)
	}
	if boolValue() {
		s := &Response{}
		p.Reply = s
		rd(&s.Sequence)
		s.Status = byteValue()
		rd(&s.Value)
		rd(&s.CheckRMSE)
		n := byteValue()
		if n > 32 {
			return Packet{}, errors.New("HNL N bound exceeded")
		}
		for i := 0; i < int(n) && err == nil; i++ {
			var u Unit
			rd(&u.Node)
			rd(&u.Code)
			rd(&u.Resolution)
			rd(&u.Contribution)
			s.Units = append(s.Units, u)
		}
		rd(&s.LearnedN)
		s.Cause = byteValue()
		s.Committed = boolValue()
		rd(&s.BeforeError)
		rd(&s.AfterError)
		rd(&s.Grown)
		rd(&s.ResolutionBefore)
		rd(&s.ResolutionAfter)
		n = byteValue()
		if n > 64 {
			return Packet{}, errors.New("HNL trial bound exceeded")
		}
		for i := 0; i < int(n) && err == nil; i++ {
			var t Trial
			t.Kind = byteValue()
			rd(&t.Node)
			rd(&t.Error)
			rd(&t.Regressions)
			t.Eligible = boolValue()
			s.Trials = append(s.Trials, t)
		}
	}
	if err != nil {
		return Packet{}, err
	}
	if m != magic || r.Len() != 0 {
		return Packet{}, errors.New("HNL unsupported magic or trailing bytes")
	}
	if err = p.Validate(); err != nil {
		return Packet{}, err
	}
	return canonicalPacket(p), nil
}
