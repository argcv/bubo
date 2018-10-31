package msg

type Operation int

const (
	OpEstablish Operation = iota + 1 // Establish
	OpRead
	OpWrite
	OpClose
	OpEstablishAck
	OpReadAck
	OpWriteAck
	OpCloseAck
	OpAliveAck // Heartbeat
)

type Meta struct {
	Op      Operation
	Wid     uint64 // Worker
	Tid     uint64 // Ticket
	Seq     uint64 // Sequence
	Status  bool   // Success or Not
	Message string
	Payload M // other payload
}

type Message struct {
	Meta Meta
	Data []byte // data stream
	Len  int
}
