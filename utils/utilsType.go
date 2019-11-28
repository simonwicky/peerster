package utils

type RumorMessageKey struct {
	Origin string
	ID uint32
}

type RumorMessages []RumorMessage

type FileInfo struct {
	Name string
	Size int64
	MetafileHash []byte
}