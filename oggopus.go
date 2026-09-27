package main

import (
	"bufio"
	"fmt"
	"io"
	"sync"
)

type OggOpusProvider struct {
	r       *bufio.Reader
	packets [][]byte
	partial []byte
	headers int
	done    chan struct{}
	once    sync.Once
}

func NewOggOpusProvider(r io.Reader) *OggOpusProvider {
	return &OggOpusProvider{
		r:    bufio.NewReader(r),
		done: make(chan struct{}),
	}
}

func (p *OggOpusProvider) ProvideOpusFrame() ([]byte, error) {
	for len(p.packets) == 0 {
		if err := p.readPage(); err != nil {
			p.finish()
			return nil, io.EOF
		}
	}
	frame := p.packets[0]
	p.packets = p.packets[1:]
	return frame, nil
}

func (p *OggOpusProvider) readPage() error {
	header := make([]byte, 27)
	if _, err := io.ReadFull(p.r, header); err != nil {
		return err
	}
	if string(header[0:4]) != "OggS" {
		return fmt.Errorf("not an ogg page")
	}

	segCount := int(header[26])
	segTable := make([]byte, segCount)
	if _, err := io.ReadFull(p.r, segTable); err != nil {
		return err
	}

	for _, segLen := range segTable {
		seg := make([]byte, segLen)
		if _, err := io.ReadFull(p.r, seg); err != nil {
			return err
		}
		p.partial = append(p.partial, seg...)

		if segLen < 255 {
			if p.headers < 2 {
				p.headers++
			} else {
				p.packets = append(p.packets, p.partial)
			}
			p.partial = nil
		}
	}
	return nil
}

func (p *OggOpusProvider) finish() {
	p.once.Do(func() { close(p.done) })
}

func (p *OggOpusProvider) Close() {
	p.finish()
}