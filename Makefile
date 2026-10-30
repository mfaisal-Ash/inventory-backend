.PHONY: swag-install swag run build

# Install CLI generator swag (sekali saja per environment/CI runner)
swag-install:
	go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Generate/refresh docs/ dari anotasi @... di komentar handler.
# WAJIB dijalankan setiap kali anotasi swag berubah, dan WAJIB dijalankan
# sebelum build pertama kali (folder docs/ belum ada di git, hasil generate).
swag:
	swag init -g cmd/main.go -o docs

run: swag
	go run ./cmd/main.go

# VERSION diambil dari tag git terbaru (kosong kalau belum ada tag -> nilai
# cadangan di appinfo_controller.go dipakai). Bisa ditimpa: make build VERSION=v1.4.0
RAW_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null)
VERSION ?= $(if $(RAW_TAG),v$(patsubst v%,%,$(RAW_TAG)))
PKG := github.com/mfaisal-Ash/inventory-backend/internal/controller/appinfo
LDFLAGS := -s -w $(if $(VERSION),-X $(PKG).CurrentVersion=$(VERSION))

build: swag
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o gowms-backend ./cmd/main.go
