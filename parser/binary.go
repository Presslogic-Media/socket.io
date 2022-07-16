package parser

import (
	"errors"
	"github.com/zishang520/engine.io/types"
	"io"
)

type Placeholder struct {
	Placeholder bool `json:"_placeholder"`
	Num         int  `json:"num"`
}

// Replaces every Buffer | ArrayBuffer | Blob | File in packet with a numbered placeholder.
func DeconstructPacket(packet *Packet) (pack *Packet, buffers []types.BufferInterface) {
	pack = packet
	pack.Data = _deconstructPacket(packet.Data, &buffers)
	pack.Attachments = uint64(len(buffers)) // number of binary 'attachments'
	return pack, buffers
}

func _deconstructPacket(data interface{}, buffers *[]types.BufferInterface) interface{} {
	if data == nil {
		return nil
	}

	if IsBinary(data) {
		_placeholder := &Placeholder{Placeholder: true, Num: len(*buffers)}
		rdata := types.NewBytesBuffer(nil)
		switch tdata := data.(type) {
		case io.Reader:
			if c, ok := data.(io.Closer); ok {
				defer c.Close()
			}
			rdata.ReadFrom(tdata)
		case []byte:
			rdata.Write(tdata)
		}
		*buffers = append(*buffers, rdata)
		return _placeholder
	} else {
		switch tdata := data.(type) {
		case []interface{}:
			newData := make([]interface{}, 0, len(tdata))
			for _, v := range tdata {
				newData = append(newData, _deconstructPacket(v, buffers))
			}
			return newData
		case map[string]interface{}:
			newData := map[string]interface{}{}
			for k, v := range tdata {
				newData[k] = _deconstructPacket(v, buffers)
			}
			return newData
		}
	}
	return data
}

// Reconstructs a binary packet from its placeholder packet and buffers
func ReconstructPacket(data *Packet, buffers []types.BufferInterface) (_ *Packet, err error) {
	data.Data, err = _reconstructPacket(data.Data, &buffers)
	data.Attachments = 0 // no longer useful
	return data, nil
}

func _reconstructPacket(data interface{}, buffers *[]types.BufferInterface) (interface{}, error) {
	if data == nil {
		return nil, nil
	}
	switch d := data.(type) {
	case []interface{}:
		newData := make([]interface{}, 0, len(d))
		for _, v := range d {
			_data, err := _reconstructPacket(v, buffers)
			if err != nil {
				return nil, err
			}
			newData = append(newData, _data)
		}
		return newData, nil
	case map[string]interface{}:
		if _placeholder, _placeholder_ok := d["_placeholder"]; _placeholder_ok {
			if placeholder, placeholder_ok := _placeholder.(bool); placeholder_ok && placeholder {
				if _num, _num_ok := d["num"]; _num_ok {
					if num, num_ok := _num.(float64); num_ok && num >= 0 && int(num) < len(*buffers) {
						return (*buffers)[int(num)], nil // appropriate buffer (should be natural order anyway)
					}
				}
				return nil, errors.New("illegal attachments")
			}
		}
		newData := map[string]interface{}{}
		for k, v := range d {
			_data, err := _reconstructPacket(v, buffers)
			if err != nil {
				return nil, err
			}
			newData[k] = _data
		}
		return newData, nil
	}
	return data, nil
}
