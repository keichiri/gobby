package announcing

type AnnounceResult struct {
	Complete   int
	Incomplete int
	PeerData   []byte
}
