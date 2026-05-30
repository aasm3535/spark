APP_NAME  := spark
BUILD_DIR := builds

VERSION   := 1.0.0
LDFLAGS   := -s -w -H=windowsgui -X main.version=$(VERSION)

.PHONY: all build clean run

all: build

# ── Full build: resources + binary ──────────────────────────
# .syso is generated in root for the linker, then cleaned up.
build: winres
	go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME).exe .
	rm -f rsrc_windows_amd64.syso

# ── Build without embedding winres (faster for dev) ────────
build-dev:
	go build -ldflags="-H=windowsgui" -o $(BUILD_DIR)/$(APP_NAME).exe .

# ── Embed Windows resources (icon, manifest, version) ──────
# go-winres generates rsrc_windows_amd64.syso in project root;
# the Go linker picks it up automatically.
winres: rsrc_windows_amd64.syso

rsrc_windows_amd64.syso: winres/winres.json
	go-winres make --product-version=$(VERSION) --file-version=$(VERSION) --arch=amd64

# ── Run the built exe ──────────────────────────────────────
run: build
	./$(BUILD_DIR)/$(APP_NAME).exe

# ── Clean build artifacts ──────────────────────────────────
clean:
	rm -rf $(BUILD_DIR)
	rm -f rsrc_windows_amd64.syso

# ── Tidy go modules ────────────────────────────────────────
tidy:
	go mod tidy
