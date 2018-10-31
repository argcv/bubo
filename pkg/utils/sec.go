package utils

import (
	"bytes"
	"crypto/rc4"
	"encoding/gob"
	"github.com/argcv/stork/log"
	"io"
	"net"
	"sync"
	"time"
)

type SecureMsgPack struct {
	Len  int
	Data []byte
}

func (p *SecureMsgPack) XORKeyStream (secret string)  {
	if len(secret) > 0 && p.Len > 0 {
		cipher, err := rc4.NewCipher([]byte(secret))

		if err != nil {
			log.Errorf("rc4.NewCipher:%v", err)
			return
		}

		cipher.XORKeyStream(p.Data[:p.Len], p.Data[:p.Len])
	}
}

type SecConn struct {
	Conn   net.Conn
	Secret string
	Buf    bytes.Buffer // read buffer

	enc *gob.Encoder
	dec *gob.Decoder

	m *sync.Mutex
}

func NewSecConn(c net.Conn, secret string) *SecConn {

	sc := &SecConn{
		Conn:   c,
		Secret: secret,
		Buf:    bytes.Buffer{},

		enc: nil,
		dec: nil,
		m:   &sync.Mutex{},
	}

	return sc
}

//func (sc *SecConn) createHash(key string) []byte {
//	hash := sha256.Sum256([]byte(key))
//	return hash[:]
//}

func (sc *SecConn) Read(b []byte) (n int, err error) {
	sc.m.Lock()
	if sc.dec == nil {
		sc.dec = gob.NewDecoder(sc.Conn)
	}
	sc.m.Unlock()

	dec := sc.dec

	// log.infof("read: %d %d", sc.Buf.Len(), len(b))
	if sc.Buf.Len() == 0 || sc.Buf.Len() < len(b)  {
		pack := &SecureMsgPack{}
		// log.Debugf("READING: %v %v", sc.Buf.Len(), len(b))

		if err := dec.Decode(pack); err == nil {

			pack.XORKeyStream(sc.Secret)

			sc.Buf.Write(pack.Data[:pack.Len])
			// log.infof("Receive: %v %v / %v, %v", pack.Len, len(pack.Data), sc.Buf.Len(), len(b))
		} else if err != io.EOF {
			// log.Debugf("ERR: %v %v / %v, %v, %v", pack.Len, len(pack.Data), sc.Buf.Len(), len(b), err)
			return 0, err
		} else {
			// log.Debugf("EOF: %v %v / %v, %v, %v", pack.Len, len(pack.Data), sc.Buf.Len(), len(b), err)
			// io.EOF, ignore
			//break
		}
	}
	i, e := sc.Buf.Read(b)
	// log.Debugf("i:%v, e:%v", i, e)
	return i, e
}

func (sc *SecConn) Write(b []byte) (n int, err error) {
	sc.m.Lock()
	if sc.enc == nil {
		sc.enc = gob.NewEncoder(sc.Conn)
	}
	sc.m.Unlock()

	dst := make([]byte, len(b))
	copy(dst, b)
	// log.infof("copy size: %v", copy(dst, b))
	pack := SecureMsgPack{
		Data: dst,
		Len:  len(dst),
	}

	pack.XORKeyStream(sc.Secret)

	// log.infof("pack: %v", string(pack.Data))
	// handle pack..

	enc := sc.enc
	if err := enc.Encode(pack); err != nil {
		// log.Debugf("Sent ERROR: %v", err)
		return 0, err
	} else {
		// log.Debugf("Sent: %v %v", pack.Len, len(pack.Data))
		return len(b), nil
	}
}

func (sc *SecConn) Close() error {
	return sc.Conn.Close()
}

func (sc *SecConn) LocalAddr() net.Addr {
	return sc.Conn.LocalAddr()
}

func (sc *SecConn) RemoteAddr() net.Addr {
	return sc.Conn.RemoteAddr()
}

func (sc *SecConn) SetDeadline(t time.Time) error {
	return sc.Conn.SetDeadline(t)
}

func (sc *SecConn) SetReadDeadline(t time.Time) error {
	return sc.Conn.SetReadDeadline(t)
}

func (sc *SecConn) SetWriteDeadline(t time.Time) error {
	return sc.Conn.SetWriteDeadline(t)
}
