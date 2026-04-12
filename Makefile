PI      := pi@raspberrypi.local
REMOTE  := /opt/huntsman-panel
BINARY  := bin/huntsman-panel
SERVICE := huntsman-panel

.PHONY: build deploy logs ssh ship

build:
	GOOS=linux GOARCH=arm64 go build -o $(BINARY) ./cmd/daemon

deploy: build
	scp $(BINARY) $(PI):$(REMOTE)/huntsman-panel
	ssh $(PI) "systemctl --user restart $(SERVICE)"

logs:
	ssh $(PI) "journalctl --user -u $(SERVICE) -f"

ssh:
	ssh $(PI)

# Full inner loop: build → push → restart → tail logs
ship: deploy logs
