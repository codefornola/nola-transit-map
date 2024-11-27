## NOLA Transit Map

Realtime map of all New Orleans public transit vehicles (streetcars and busses). You can view the map here for the time being: [https://nolatransit.fly.dev/](https://nolatransit.fly.dev/)

For some reason, going from the old RTA app to the new Le Pass app has resulted in the loss of realtime functionality. Not having realtime makes
getting around the city extremely frustrating. This map is just a stop gap to get the data out to people again.

### Needs to be Done

* somehow communicate the staleness of the data to the user (we have a status indicator for the connection but could use the vehicle timestamps)
* nice icons and popups for the vehicles
* show the user's location on the map

### Contributing

Join #civic-hacking in Nola Devs Slack channel, where this project is discussed: https://nola.slack.com/join/shared_invite/zt-4882ja82-iGm2yO6KCxsi2aGJ9vnsUQ.

The API key is coveted. Only a few chosen ones will be granted access, hence the included mock server that is used in DEV mode.

You need a few things on your machine to build the project. If you are an `asdf` user there is a .tool-versions file with acceptable versions of node, npm, and go, but not make to keep from conflicting with system build tools.

* node and npm
* go
* make

### To Run in Development:

1. Run the mock bustime server in a terminal. The mock server serves fake vehicle data. Vehicles will appear stationary.
    ```
    # terminal tab 1 - Mock bustime server
    make mock
    ```

2. Run the main server _in another terminal_.
    ```
    # terminal tab 2 - Go backend
    make dev DEV=1
    ```

3. (Optional) If working on the frontend, you probably want changes to trigger a build automatically. You'll still have to refresh the page to see changes. 
_In a 3rd terminal_:
    ```
    # terminal tab 3 - React frontend
    make watch
    ```

4. Open the frontend [http://localhost:8080](http://localhost:8080)

    You may need to refresh the page after the browser window is automatically opened by the `make` command.

### To Run in Production:

Add the API and IP env vars to `make run`:
```
make build && make run CLEVER_DEVICES_KEY=thekey CLEVER_DEVICES_IP=ipaddr
```