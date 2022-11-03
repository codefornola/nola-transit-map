CLEVER_DEVICES_KEY?=""
CLEVER_DEVICES_IP?=""

build:
	npm run build
	go build

run:
	./nola-transit-map

show:
	open http://localhost:8080

watch:
	npm run watch
