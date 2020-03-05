package api

import (
	"github.com/vitelabs/go-vite/vite"
)

type UtilApi struct {
	vite *vite.Vite
}

func NewUtilApi(vite *vite.Vite) *UtilApi {
	return &UtilApi{
		vite: vite,
	}
}
