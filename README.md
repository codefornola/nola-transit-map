## NOLA Transit Map

Realtime map of all New Orleans public transit vehicles (streetcars and busses). You can view the map here for the time being: [https://nolatransit.fly.dev/](https://nolatransit.fly.dev/)

For some reason, going from the old RTA app to the new Le Pass app has resulted in the loss of realtime functionality. Not having realtime makes
getting around the city extremely frustrating. Fortunately I reverse engineered the old app a few years ago and have access to the raw real-time stream.
This map is just a stop gap to get the data out to people again.

### Needs to be done

* somehow communicate the staleness of the data to the user (we have a status indicator for the connection but could use the vehicle timestamps)
* nice icons and popups for the vehicles
* show the user's location on the map

### Developing

You need to get the key from Ben on Slack in #civic-hacking. If you aren't in the NOLA Devs slack here is an invite: https://nola.slack.com/join/shared_invite/zt-4882ja82-iGm2yO6KCxsi2aGJ9vnsUQ

You need a few things deps your machine:

* node and npm
* go
* make

First build. This builds the frontend application and compiles the go code:

```
make build
```

To run:

```
make run CLEVER_DEVICES_KEY=thekey
```

Open the frontend [http://localhost:8080](http://localhost:8080)


You need to run `make build` (or `npm run build`) to build the frontend code. It doesn't auto-build. You can run an auto-build in a new tab:

```
make watch
```

but you still need to refresh the page