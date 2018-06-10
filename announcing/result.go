package announcing

type AnnounceResult struct {
	Complete   int32
	Incomplete int32
	PeerData   []byte
}
