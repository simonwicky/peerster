package utils

import ("fmt")

type RumorKeyQueue struct {
	Container []RumorMessageKey
	size uint32
	currentSize uint32
}

func NewRumorKeyQueue(n uint32) *RumorKeyQueue {
	return &RumorKeyQueue{
		Container: []RumorMessageKey{},
		size : n,
		currentSize : 0,
	}
}

func (r *RumorKeyQueue) Push(key RumorMessageKey) {
	r.Container = append(r.Container, key)
	r.currentSize += 1
	if r.currentSize > r.size {
		r.Container = r.Container[1:]
		r.currentSize -= 1
		fmt.Println(r.Container)
	}
	fmt.Println(r.Container)
	fmt.Println(r.currentSize)
	fmt.Println(r.size)
	return
}


