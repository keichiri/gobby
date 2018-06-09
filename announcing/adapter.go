package announcing

type trackerAdapter interface {
	Announce(map[string]interface{}) (*AnnounceResult, int, error)
}
