.PHONY: dirs

all: android ledsvc

android:
	fyne-cross android \
		-app-id com.luksamuk.ledcontrol \
		-icon Icon.png

ledapp:
	go build -o bin/ledapp main.go

ledsvc: dirs
	CGO_ENABLED=0 go build -o bin/ledsvc cmd/ledsvc/ledsvc.go

container-push:
	docker buildx build \
		-f Dockerfile \
		--platform=linux/amd64,linux/arm64 \
		-t luksamuk/ledsvc:latest \
		--push \
		.

dirs:
	@mkdir -p bin/
