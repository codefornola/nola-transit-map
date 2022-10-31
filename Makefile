CLEVER_DEVICES_KEY?=""

build:
	npm run build
	go build

run:
	go run main.go

show:
	open http://localhost:8080

watch:
	npm run watch
