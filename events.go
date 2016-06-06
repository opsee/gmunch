package gmunch

import (
	"bytes"
	"encoding/gob"
)

type Decoder interface {
	Decode(interface{}) error
}

func (event *Event) EncodeData(data interface{}) error {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(data)
	if err != nil {
		return err
	}
	event.Data = b.Bytes()
	return nil
}

func (event *Event) Decoder() Decoder {
	return gob.NewDecoder(bytes.NewBuffer(event.Data))
}
