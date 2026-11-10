/**
 * @file emit.go
 * @author Vladyslav Aviedov <vladaviedov at protonmail dot com>
 * @version 0.9.1
 * @date 2026
 * @license GPLv3.0
 * @brief Output data manager.
 */
package main

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

type Emitter struct {
	InMemBuffer bytes.Buffer
	SpillFile   *os.File

	IMReader     *bufio.Reader
	OutputStream io.Writer
}

func newEmitter(stream io.Writer) *Emitter {
	return &Emitter{
		OutputStream: stream,
	}
}

func (e *Emitter) writeIM(data string) error {
	if e.SpillFile != nil {
		_, err := e.SpillFile.WriteString(data)
		return err
	}

	if e.InMemBuffer.Len()+len(data) <= int(opts.IMBufferSize) {
		_, err := e.InMemBuffer.WriteString(data)
		return err
	}

	// Create spill file
	tempFile, err := os.CreateTemp("", "shpp-imbuf-*")
	if err != nil {
		return err
	}
	e.SpillFile = tempFile

	// Move in-memory contents to file
	_, err = tempFile.Write(e.InMemBuffer.Bytes())
	if err != nil {
		return err
	}

	// Write new data
	_, err = tempFile.WriteString(data)
	if err != nil {
		return err
	}

	// Release memory
	e.InMemBuffer = bytes.Buffer{}
	return nil
}

func (e *Emitter) getIMReader() *bufio.Reader {
	if e.IMReader == nil {
		if e.SpillFile != nil {
			e.SpillFile.Seek(0, io.SeekStart)
			e.IMReader = bufio.NewReader(e.SpillFile)
		} else {
			e.IMReader = bufio.NewReader(&e.InMemBuffer)
		}
	}

	return e.IMReader
}

func (e *Emitter) writeOut(data string) error {
	_, err := e.OutputStream.Write([]byte(data))
	return err
}

func (e *Emitter) flushIM() error {
	_, err := io.Copy(e.OutputStream, e.getIMReader())
	return err
}

func (e *Emitter) destroy() {
	if e.SpillFile == nil {
		return
	}

	e.SpillFile.Close()
	os.Remove(e.SpillFile.Name())
}
