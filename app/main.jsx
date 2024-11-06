import React, { useRef, useEffect, forwardRef } from 'react'
import { createRoot } from 'react-dom/client';
import { MapContainer, TileLayer, Marker, Popup, GeoJSON } from 'react-leaflet'
import { BsInfoLg, BsFillCircleFill, BsFillCloudSlashFill, BsFillExclamationTriangleFill } from 'react-icons/bs'
import L from 'leaflet';
import "leaflet-rotatedmarker";
import 'bootstrap/dist/css/bootstrap.min.css';
import NortaGeoJson from '../data/routes.json';
import Row from 'react-bootstrap/Row';
import Col from 'react-bootstrap/Col';
import Select from 'react-select'
import makeAnimated from 'react-select/animated';
import CustomModal from './components/modal';
import LocationMarker from './components/location';
import './main.css';

const animatedComponents = makeAnimated();
const ROUTES = NortaGeoJson
    .features
    .filter(f => f.geometry.type === "MultiLineString" && f.properties.route_id)
    .reduce((acc, f) => {
        return {
            ...acc,
            [f.properties.route_id]: <GeoJSON key={f.properties.route_id} data={f} pathOptions={{ color: f.properties.route_color }} />
        }
    }, {})

const iconVehicle = new L.Icon({
    iconUrl: require('../img/arrow.png'),
    iconRetinaUrl: require('../img/arrow.png'),
    iconAnchor: null,
    shadowUrl: null,
    shadowSize: null,
    shadowAnchor: null,
    iconSize: new L.Point(24, 24),
    className: 'leaflet-marker-icon'
});

const RotatedMarker = forwardRef(({ children, ...props }, forwardRef) => {
    const markerRef = useRef();

    const { rotationAngle, rotationOrigin } = props;
    useEffect(() => {
        const marker = markerRef.current;
        if (marker) {
            marker.setRotationAngle(rotationAngle);
            marker.setRotationOrigin(rotationOrigin);
        }
    }, [rotationAngle, rotationOrigin]);

    return (
        <Marker
            ref={(ref) => {
                markerRef.current = ref;
                if (forwardRef) {
                    forwardRef.current = ref;
                }
            }}
            {...props}
        >
            {children}
        </Marker>
    );
});

function timestampDisplay (timestamp) {
    const relativeTimestamp = new Date() - new Date(timestamp);
    if (relativeTimestamp < 60000) { return 'less than a minute ago'; }
    const minutes = Math.round(relativeTimestamp / 60000);
    if (minutes === 1) { return '1 minute ago'}
    return minutes + ' minutes ago';
}


class App extends React.Component {
    constructor(props) {
        super(props)
        const routes = localStorage.getItem("routes") || "[]"
        this.state = {
            vehicles: [],
            routes: JSON.parse(routes),
            connected: false,
            lastUpdate: new Date(),
            now: new Date(),
            websocket: null,
        }
        this.handleRouteChange = this.handleRouteChange.bind(this)
    }

    componentWillMount() {
        this.connectWebSocket();
        this.interval = setInterval(() => this.setState({ now: Date.now() }), 1000);
    }

    componentWillUnmount() {
        this.closeWebSocket();
        clearInterval(this.interval)
    }

    connectWebSocket = () => {
        const scheme = window.location.protocol == "http:" ? "ws" : "wss"
        const url = `${scheme}://${window.location.hostname}:${window.location.port}/ws`
        const websocket = new WebSocket(url);

        websocket.onopen = () => {
            console.log('Websocket connected');
            this.setState({ connected: true });
        };

        websocket.onmessage = (evt) => {
            console.log('WebSocket message');
            if (!this.state.connected) this.setState({ connected: true })
            const vehicles = JSON.parse(evt.data)
            const lastUpdate = new Date()
            this.setState({
                vehicles,
                lastUpdate,
            })
            console.dir(vehicles)
        };

        websocket.onclose = () => {
            console.log('WebSocket closed');
            this.setState({ connected: false });

            setTimeout(this.connectWebSocket, 5000);
        };

        websocket.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.setState({ connected: false });

            setTimeout(this.connectWebSocket, 5000);
        };

        this.setState({ websocket });
    };

    closeWebSocket = () => {
        if (this.state.websocket) {
            this.state.websocket.close();
        }
    };

    routeComponents() {
        if (this.state.routes.length === 0) return Object.values(ROUTES)

        return this.state.routes
            .map(r => r.value)
            .map(rid => ROUTES[rid])
            .filter(r => r !== null)
    }

    markerComponents() {
        let query = (_v) => true
        if (this.state.routes.length > 0) {
            const values = this.state.routes.map(r => r.value)
            query = (v) => values.includes(v.rt)
            console.log("query filter on routes: " + values)
        }

        return this.state.vehicles
            .filter(query)
            .map(v => {
                const coords = [v.lat, v.lon].map(parseFloat)
                const rotAng = parseInt(v.hdg, 10)
                const relTime = timestampDisplay(v.tmstmp)
                return <RotatedMarker key={v.vid} position={coords} icon={iconVehicle} rotationAngle={rotAng} rotationOrigin="center">
                    <Popup>
                        {v.rt}{v.des ? ' - ' + v.des : ''}
                        <br/>{relTime}
                    </Popup>
                </RotatedMarker>
            })
    }

    mapContainer() {
        return <MapContainer center={[29.95569, -90.0786107]} zoom={13} zoomControl={false}>
            <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
            />
            {this.markerComponents()}
            {this.routeComponents()}
            <LocationMarker />
        </MapContainer>
    }

    notConnectedScreen() {
        return <Row className="justify-content-md-center">
            <Col md="auto">
                <p>Connection broken. Attempting to reconnect...</p>
            </Col>
        </Row>
    }

    buildControlBar() {
        let connectionStatus = this.state.connected 
            ? <React.Fragment>
                <span className="control-bar__connection-container connected"><BsFillCircleFill /><span class="control-bar__label-text">Connected</span></span>
              </React.Fragment> 
            : <React.Fragment>
                <span className="control-bar__connection-container not-connected"><BsFillCloudSlashFill /><span class="control-bar__label-text">Not Connected</span></span>
              </React.Fragment>

        if (this.state.connected && this.lagging()) connectionStatus = 
            <React.Fragment>
                <span className="control-bar__connection-container trouble-connecting"><BsFillExclamationTriangleFill />Trouble Connecting...</span>
            </React.Fragment>

        if (!this.state.connected) return this.notConnectedScreen()

        if (this.state.vehicles.length === 0) {
            return <Row className="justify-content-md-center">
                <Col md="auto">
                    <p>No Vehicles found yet. Are you connected?</p>
                </Col>
            </Row>
        }

        const routes = [...new Set(this.state.vehicles.map(v => v.rt))]
        const routeOptions = routes.map(r => {
            return { value: r, label: r }
        })

        return <div class="control-bar">
                    <label class="control-bar__filter-label"><span class="control-bar__label-text">Filter Routes:</span>
                        <Select
                            closeMenuOnSelect={false}
                            components={animatedComponents}
                            defaultValue={[]}
                            value={this.state.routes}
                            isMulti
                            options={routeOptions}
                            onChange={this.handleRouteChange}
                            className="route-filter"
                            placeholder="Filter Select Route(s)"
                        />
                    </label>
                    {connectionStatus}
        </div>
    }

    handleRouteChange(routes) {
        this.setState({ routes })
        localStorage.setItem("routes", JSON.stringify(routes))
    }

    lagging() {
        // lagging by over 13 seconds
        return Math.floor((this.state.now - this.state.lastUpdate) / 1000) > 13
    }

    render() {
        return <div className="App">
            <main>
                {this.buildControlBar()}
                {this.mapContainer()}
                <CustomModal
                    title='NOLA Transit Map'
                    subtitle={['Created by ', <a href="https://codeforneworleans.org/"> Code For New Orleans</a>]}
                    buttonText={[<BsInfoLg />, 'About this project']}
                    content={
                        [
                            'When the RTA switched to the new LePass app, all of the realtime data stopped working. Relying on public transportation in New Orleans without this data is extremely challenging. We made this map as a stop gap until realtime starts working again.',

                            <br></br>,<br></br>, 'If you find a problem, or have a feature request, consider ', <a href="https://github.com/codefornola/nola-transit-map/issues">filing an issue here.</a>,
                            ' You can also join us on slack in the #civic-hacking channel of the ', <a href="https://join.slack.com/t/nola/shared_invite/zt-4882ja82-iGm2yO6KCxsi2aGJ9vnsUQ">Nola Devs slack.</a>,
                            ' Take a look at ', <a href="https://github.com/codefornola/nola-transit-map">the README on GitHub</a>, ' to learn more about how it works.'
                        ]
                    }
                />
            </main>
        </div>
    }
}

window.initApp = function () {
    console.log("initApp()")

    const root = createRoot(
        document.getElementById('main')
    )

    root.render(<App />)
}
