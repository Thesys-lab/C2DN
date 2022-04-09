package main

import (
	"bufio"
	"encoding/binary"
)

type Request struct {
	Timestamp uint32
	ID        uint32
	Size      uint32
}

func readOneReq(reader *bufio.Reader) (Request, error) {
	var req Request

	err := binary.Read(reader, binary.LittleEndian, &req)
	if err != nil {
		return req, err
	}

	return req, err
}

