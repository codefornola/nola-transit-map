build:
	npm install
	npm run build
	go build

run:
	./nola-transit-map

show:
	open http://localhost:8080

### DEV ###

mock:
	cd mock_bustime_server && go build && ./server
	
dev:
	npm install
	go build
	sh -c 'DEV=1 CLEVER_DEVICES_KEY=1 CLEVER_DEVICES_IP=1 ./nola-transit-map' || echo "Couldn't run the binary."

watch:
	npm run watch