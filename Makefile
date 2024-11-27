CLEVER_DEVICES_KEY?=""
CLEVER_DEVICES_IP?=""

build:
	npm install
	npm run build
	go build

run:
	./nola-transit-map

show:
	open http://localhost:8080

### DEV ###
export CLEVER_DEVICES_KEY := DEV
export CLEVER_DEVICES_IP := DEV

mock:
	cd mock_bustime_server && go build && ./server
	
dev:
	npm install
	go build
	./nola-transit-map || echo "Couldn't run the binary."

watch:
	npm run watch