package api

import (
	"github.com/vitelabs/go-vite/vite"
)

type Pow struct {
	vite *vite.Vite
}

func NewPow(vite *vite.Vite) *Pow {
	return &Pow{
		vite: vite,
	}
}
