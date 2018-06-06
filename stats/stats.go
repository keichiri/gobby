package stats

type Stats struct{}

type CurrentStats struct {
	Downloaded int
	Uploaded   int
	Left       int
}

func (s *Stats) GetCurrent() *CurrentStats {
	return &CurrentStats{}
}
