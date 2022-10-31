import React from 'react'
import {createRoot} from 'react-dom/client';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet'


class MainMap extends React.Component {
    constructor(props) {
        super(props)
        this.state = {
            vehicles: [],
            connected: false,
        }
    }

    componentDidMount() {
        const scheme = window.location.protocol == "http:" ? "ws" : "wss"
        const url = `${scheme}://${window.location.hostname}:${window.location.port}/ws`
        const conn = new WebSocket(url);
        conn.onclose = () => {
            console.log("Closing websocket")
            this.setState({connected: false})
        }
        conn.onmessage = (evt) => {
            console.log('onmessage');
            if (!this.state.connected) this.setState({connected: true})
            const vehicles = JSON.parse(evt.data)
            console.dir(vehicles)
            this.setState({vehicles})
        }
    }

    markers() {
        return this.state.vehicles
            .map(v => {
                const coords = [v.lat, v.lon].map(parseFloat) 
                return <Marker key={v.vid} position={coords}>
                    <Popup>
                        { v.rt + " - " + v.des + " - " + v.tmstmp}
                    </Popup>
                </Marker>
            })
    }

    render() {
        return <MapContainer center={[29.95569, -90.0786107]} zoom={13}>
            <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                url="https://tile.openstreetmap.org/{z}/{x}/{y}.png"
            />

            {this.markers()}
        </MapContainer>
    }
}

window.initApp = function () {
    console.log("initApp()")

    const root = createRoot(
        document.getElementById('main')
        )

    root.render(<MainMap />)
}
