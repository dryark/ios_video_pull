package main

import (
    "bytes"
    "encoding/binary"
    "io"
    //"fmt"
    "strconv"
    "time"
    cm "github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
    
    "go.nanomsg.org/mangos/v3"
    // register transports
	  _ "go.nanomsg.org/mangos/v3/transport/all"
)

var startCode = []byte{00, 00, 00, 01}

//ZMQWriter writes nalus into a file using 0x00000001 as a separator (h264 ANNEX B) and raw pcm audio into a wav file
type NanoWriter struct {
    socket mangos.Socket
    buffer bytes.Buffer
    fh     io.Writer
}

func NewNanoWriter( socket mangos.Socket ) NanoWriter {
    return NanoWriter{ socket: socket }
}

func NewFileWriter( fh io.Writer ) NanoWriter {
    return NanoWriter{ fh: fh }
}

//Consume writes PPS and SPS as well as sample bufs into a annex b .h264 file and audio samples into a wav file
func (avfw NanoWriter) Consume(buf cm.CMSampleBuffer) error {
    if buf.MediaType == cm.MediaTypeSound {
        return avfw.consumeAudio(buf)
    }
    return avfw.consumeVideo(buf)
}

func (self NanoWriter) Stop() {}

func (self NanoWriter) consumeVideo(buf cm.CMSampleBuffer) error {
    if buf.HasFormatDescription {
        err := self.writeNalu(buf.FormatDescription.PPS)
        if err != nil {
            return err
        }
        err = self.writeNalu(buf.FormatDescription.SPS)
        if err != nil {
            return err
        }
    }
    if !buf.HasSampleData() {
        return nil
    }
    return self.writeNalus(buf.SampleData)
}

func (self NanoWriter) writeNalus(bytes []byte) error {
    slice := bytes
    for len(slice) > 0 {
        length := binary.BigEndian.Uint32(slice)
        err := self.writeNalu(slice[4 : length+4])
        if err != nil {
            return err
        }
        slice = slice[length+4:]
    }
    return nil
}

func (self NanoWriter) writeNalu(naluBytes []byte) error {
    now := strconv.FormatInt( time.Now().UnixNano()/1000000, 10 )
    json := "{\"nalBytes\":" + strconv.Itoa( len( naluBytes ) + 4 ) + ",\"time\":" + now + "}";
    
    var jsonLen uint16 = uint16( len( json ) )
    binary.Write( &self.buffer, binary.LittleEndian, jsonLen )
    self.buffer.Write( []byte( json ) )
    
    _, err := self.buffer.Write(startCode)
    if err != nil {
        return err
    }
    _, err = self.buffer.Write(naluBytes)
    if err != nil {
        return err
    }
    
    if self.fh == nil {
        self.socket.Send( self.buffer.Bytes() )
    } else {
        self.fh.Write( self.buffer.Bytes() )
    }
    self.buffer.Reset()
    return nil
}

func (self NanoWriter) consumeAudio(buffer cm.CMSampleBuffer) error {
    return nil
}
