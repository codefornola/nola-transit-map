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

import vehicleTypes from '../data/vehicle_types.json'
import busIcon from '../img/icon-mock-bus.png'
import streetcarIcon from '../img/icont-mock-streetcar.png'
import ferryIcon from '../img/icon-mock-ferry.png'
import errorIcon from '../img/icon-vehicle-error.png'
import arrowIcon from '../img/arrow_offset.png'

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

// These routes don't exist at NORTA.com
// When a vehicle enters its garage, its route becomes 'U'
// The definition of PO and PI routes is unknown -> filter out for now
const NOT_IN_SERVICE_ROUTES = ['U', 'PO', 'PI']

const MARKER_ICON_SIZE = 24

const iconArrow = new L.Icon({
    iconUrl: arrowIcon,
    iconRetinaUrl: arrowIcon,
    // Tall so arrow doesn't intersect vehicle (& match aspect ratio of graphic)
    iconSize: [MARKER_ICON_SIZE, MARKER_ICON_SIZE * 2],
    className: 'leaflet-marker-icon'
});

function ArrowMarker(props) {
    const { rotationAngle } = props;
    const markerRef = useRef();
    useEffect(() => {
        markerRef.current?.setRotationAngle(rotationAngle);
    }, [rotationAngle]);
    return <Marker ref={markerRef} {...props} icon={iconArrow} rotationOrigin="center" />;
}

function vehicleIcon(image) {
    return new L.Icon({
        iconUrl: image,
        iconRetinaUrl: image,
        iconSize: [MARKER_ICON_SIZE, MARKER_ICON_SIZE],
        className: 'leaflet-marker-icon'
    });
}

const vehicleIcons = Object.freeze({
    ferry: vehicleIcon(ferryIcon),
    streetcar: vehicleIcon(streetcarIcon),
    bus: vehicleIcon(busIcon),
    error: vehicleIcon(errorIcon),
})

function VehicleMarker({ children, ...props }) {
    const { type } = props
    return (
        <Marker {...props} icon={vehicleIcons[type]} riseOnHover={true}>
            {children}
        </Marker>
    )
}

function timestampDisplay(timestamp) {
    const relativeTimestamp = new Date() - new Date(timestamp);
    if (relativeTimestamp < 60000) { return 'less than a minute ago'; }
    const minutes = Math.round(relativeTimestamp / 60000);
    if (minutes === 1) { return '1 minute ago'}
    return minutes + ' minutes ago';
}

const scheme = window.location.protocol == "http:" ? "ws" : "wss"
const url = `${scheme}://${window.location.hostname}:${window.location.port}/ws`
const conn = new WebSocket(url);

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
        }
        this.handleRouteChange = this.handleRouteChange.bind(this)
    }

    componentWillMount() {
        conn.onopen = () => {
            console.log("Websocket Open")
            this.setState({ connected: true })
        }
        conn.onclose = () => {
            console.log("Closing websocket")
            this.setState({ connected: false })
        }
        conn.onmessage = (evt) => {
            console.log('onmessage');
            if (!this.state.connected) this.setState({ connected: true })
            const vehicles = JSON.parse(evt.data)
                // filter out unused vehicles
                .filter(v => !NOT_IN_SERVICE_ROUTES.includes(v.rt))
            const lastUpdate = new Date()
            this.setState({
                vehicles,
                lastUpdate,
            })
            // console.dir(vehicles)
        }
        this.interval = setInterval(() => this.setState({ now: Date.now() }), 1000);
    }

    componentWillUnmount() {
        clearInterval(this.interval)
    }

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
                const type = vehicleTypes[v.rt]?.type ?? 'error'
                return (
                    <div key={v.vid + '_container'}>
                        <ArrowMarker key={v.vid + '_arrow'} position={coords} rotationAngle={rotAng} />
                        <VehicleMarker key={v.vid} type={type} position={coords}>
                            <Popup>
                                {v.rt}{v.des ? ' - ' + v.des : ''}
                                <br/>
                                {relTime}
                            </Popup>
                        </VehicleMarker>
                    </div>
                )
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
                <p>Looks like you aren't connected. Maybe try refreshing the page. If it's not working please <a href="https://github.com/codefornola/nola-transit-map/issues">get in touch with us</a>.</p>
            </Col>
        </Row>
    }

    buildControlBar() {
        let connectionStatus = this.state.connected 
            ? <React.Fragment>
                <span className="control-bar__connection-container connected"><BsFillCircleFill /><span className="control-bar__label-text">Connected</span></span>
              </React.Fragment> 
            : <React.Fragment>
                <span className="control-bar__connection-container not-connected"><BsFillCloudSlashFill /><span className="control-bar__label-text">Not Connected</span></span>
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

        return <div className="control-bar">
                    <label className="control-bar__filter-label"><span className="control-bar__label-text">Filter Routes:</span>
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
