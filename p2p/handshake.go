package p2p

type HandshakerFunc func(Peer) error

func NOPHandshakerFunc(Peer) error {
	return nil
}
