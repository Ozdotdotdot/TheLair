PI      := pi@raspberrypi.local
REMOTE  := /opt/huntsman-panel
BINARY  := bin/huntsman-panel
SERVICE := huntsman-panel

.PHONY: build deploy logs ssh ship

build:
	GOOS=linux GOARCH=arm64 go build -o $(BINARY) ./cmd/daemon

deploy: build
	ssh $(PI) "systemctl --user stop $(SERVICE)"
	scp $(BINARY) $(PI):$(REMOTE)/huntsman-panel
	ssh $(PI) "systemctl --user start $(SERVICE)"

logs:
	ssh $(PI) "journalctl --user -u $(SERVICE) -f"

ssh:
	ssh $(PI)

# Full inner loop: build → push → restart → tail logs
ship: deploy logs
