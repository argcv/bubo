package utils

import (
	"encoding/gob"
	"fmt"
	"github.com/argcv/picidae/pkg/msg"
	"io/ioutil"
	"net"
	"sync"
	"testing"
)

func TestNewSecureConn(t *testing.T) {
	listenStr := "127.0.0.1:19938"
	secret := "this is a secret"
	text := "my text is a secret, my text is a secret"

	go func() {
		conn, err := net.Dial("tcp", listenStr)
		if err != nil {
			t.Fatal(err)
		}
		seConn := NewSecConn(conn, secret)
		defer seConn.Close()

		if _, err := fmt.Fprintf(seConn, text); err != nil {
			t.Fatal(err)
		}

	}()

	l, err := net.Listen("tcp", listenStr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		seConn := NewSecConn(conn, secret)
		defer seConn.Close()

		buf, err := ioutil.ReadAll(seConn)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(string(buf[:]))

		if text2 := string(buf[:]); text2 != text {
			t.Fatalf("Unexpected message:\nGot:\t\t%s\nExpected:\t%s\n", text2, text)
		}
		break
	}
}

func TestNewSecureConn2(t *testing.T) {
	listenStr := "127.0.0.1:19939"
	secret := "this is a secret"

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		conn, err := net.Dial("tcp", listenStr)
		if err != nil {
			t.Fatal(err)
		}
		seConn := NewSecConn(conn, secret + "#")
		encoder := gob.NewEncoder(seConn)
		decoder := gob.NewDecoder(seConn)

		message := msg.Message{
			Op:   msg.OpEstablish,
			Data: []byte("abcdefg"),
			Tid:  123456,
		}
		for i := 0; i < 10; i ++ {
			message.Tid += 1
			tk := message.Tid
			if err := encoder.Encode(message); err != nil {
				t.Fatal(err)
			} else {
				t.Logf("encoder.Encode: written")
			}
			if err := decoder.Decode(&message); err != nil {
				t.Fatal(err)
			} else {
				t.Logf("encoder.Encode: written")
			}
			t.Logf("tk: %v => %v", tk, message.Tid)
		}
		seConn.Close()
		wg.Done()
	}()

	l, err := net.Listen("tcp", listenStr)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	conn, err := l.Accept()
	if err != nil {
		return
	}
	seConn := NewSecConn(conn, secret + "#")
	defer seConn.Close()

	encoder := gob.NewEncoder(seConn)
	decoder := gob.NewDecoder(seConn)
	for i := 0; i < 10; i ++ {

		newMessage := msg.Message{}

		t.Log("reading..")
		if err := decoder.Decode(&newMessage); err != nil {
			t.Fatal(err)
		} else {
			t.Logf("decoder.Decode: gottchar: %v, %v, %v", newMessage.Op, newMessage.Tid, string(newMessage.Data))
		}

		newMessage.Tid ++

		if err := encoder.Encode(newMessage); err != nil {
			t.Fatal(err)
		} else {
			t.Logf("encoder.Encode: gottchar: %v, %v, %v", newMessage.Op, newMessage.Tid, string(newMessage.Data))
		}
	}

	wg.Wait()
}
