package monitarr

type MonitarrContainer struct {
	Name string
	Id   string
}

type MonitarrLog struct {
	data          []byte
	containerName string
}
