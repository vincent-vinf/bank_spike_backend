package filter

import (
	"log"
	"testing"
)

type S struct {
	ss *SS
}

type SS struct {
	name string
}

func TestFilter(t *testing.T) {
	s1 := &SS{name: "1"}
	*s1 = SS{name: "2"}
	log.Println(s1)
}
