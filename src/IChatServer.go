package src

type IChatServer interface {
	Open(port int) error
	Broadcast(msg IMsg)

	Close()
	GetLogs() []string
}
