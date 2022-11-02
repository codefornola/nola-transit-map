import React, {useRef, useEffect, forwardRef} from 'react'
import {createRoot} from 'react-dom/client';
import { MapContainer, TileLayer, Marker, Popup} from 'react-leaflet'
import L from 'leaflet';
import "leaflet-rotatedmarker";

const iconVehicle = new L.Icon({
    iconUrl: require('../img/arrow.svg'),
    iconRetinaUrl: require('../img/arrow.svg'),
    iconAnchor: null,
    popupAnchor: null,
    shadowUrl: null,
    shadowSize: null,
    shadowAnchor: null,
    iconSize: new L.Point(24, 24),
    className: 'leaflet-div-icon'
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
                const rotAng = parseInt(v.hdg, 10)
                return <RotatedMarker key={v.vid} position={coords} icon={iconVehicle} rotationAngle={rotAng} rotationOrigin="center">
                    <Popup>
                        { v.rt + " - " + v.des + " - " + v.tmstmp}
                    </Popup>
                </RotatedMarker>
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
