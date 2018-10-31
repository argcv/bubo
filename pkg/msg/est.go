package msg

type EstablishRequest struct {
	Wid uint64
}

type EstablishAck struct {
	Staus   bool
	Wid     uint64
	Message string
}

