// Package humancheck mengekspos endpoint untuk mengambil token verifikasi
// "human check" self-hosted (lihat pkg/humancheck untuk implementasi
// issue/verify token-nya).
package humancheck

import "github.com/projsonal/gowms/pkg/humancheck"

// Controller menangani endpoint HTTP untuk menerbitkan token human-check.
type Controller struct {
	svc *humancheck.Service
}

// New membuat instance Controller HumanCheck.
func New(svc *humancheck.Service) *Controller {
	return &Controller{svc: svc}
}
