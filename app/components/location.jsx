import React, { useState, useEffect } from 'react';
import L from 'leaflet';
import { useMap } from 'react-leaflet'; 
import { CircleMarker, Popup } from 'react-leaflet';

export default function LocationMarker() {
    const [position, setPosition] = useState(null);
    const [bbox, setBbox] = useState([]);

    const map = useMap();

    useEffect(() => {
      map.locate().on("locationfound", function (e) {
        setPosition(e.latlng);
        map.flyTo(e.latlng, map.getZoom());
        const radius = e.accuracy;
        const circle = L.circle(e.latlng, radius);
        circle.addTo(map);
        setBbox(e.bounds.toBBoxString().split(","));
      });
    }, [map]);

    return position === null ? null : (
      <CircleMarker center={position}>
        <Popup>
          You are here.
        </Popup>
      </CircleMarker>
    );
  }