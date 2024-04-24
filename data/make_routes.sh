#!/bin/sh

#init list of routes
jq . > routes.json << EOF
{
"type": "FeatureCollection",
"name": "routes",
"crs": { "type": "name", "properties": { "name": "urn:ogc:def:crs:OGC:1.3:CRS84" } },
"features": []
}
EOF

for route in 12 46 47 48 49 1 4 3 8 9 11 27 31 32 45 51 52 53-O 55 57 61 62 62-O 66 67 68 80 84 86 91 103 105 114A 114B 201 202; do

  echo "ROUTE $route"

  #get stops and lines for each route direction
  for dir in 0 1; do
    wget -q -O route_${route}_dir${dir} "https://www.norta.com/RTAGetRoute?routeID=${route}&directionID=${dir}"
    cat route_${route}_dir${dir} | jq -c '{type: "MultiPoint", coordinates: .[0].stops | [.[] | [.stopLongitude, .stopLatitude]]}' > route_${route}_dir${dir}_stops
    cat route_${route}_dir${dir} | jq -c '{type: "MultiLineString", coordinates: [.[0].lines[0].latLongList | [.[] | [.shapeLog, .shapeLat]]]}' > route_${route}_dir${dir}_lines

    #fix bogus RTA datapoints
    if [ "$route" = "57" ] && [ "$dir" = "0" ]; then
      sed -i -e 's/\"-90.054342\",\"29.968979\"/\"-90.05486\",\"29.97270\"/g' route_${route}_dir${dir}_stops
      sed -i -e 's/\[\"-90.055450\",\"29.972701\"\].*\[\"-90.052978\",\"29.972807\"\]/\[\"-90.05486\",\"29.97270\"\]/g' route_${route}_dir${dir}_lines
    fi
  done

  #remove duplicate second direction info if identical to first
  if cmp --silent -- "route_${route}_dir0_lines" "route_${route}_dir1_lines"; then
    echo "" > route_${route}_dir1_lines
  fi
  if cmp --silent -- "route_${route}_dir0_stops" "route_${route}_dir1_stops"; then
    echo "" > route_${route}_dir1_stops
  fi

  #for header just use direction0 to get route name, etc
  cat route_${route}_dir0 | jq '.[0] | {type: "Feature", properties: { agency_name: "NORTA", route_id: .routeCode, agency_id: "1", route_short_name: .routeCode, route_long_name: .routeName, route_type: 3, route_color: ("#" + .routeColor), route_text_color: "#000000" }, geometry: {type: "GeometryCollection", geometries: []}}' > route_${route}_header

  #combine stops and lines into geometrycollection
  jq '.geometry.geometries += $stops0 + $stops1 + $lines0 + $lines1' route_${route}_header --slurpfile stops0 route_${route}_dir0_stops --slurpfile stops1 route_${route}_dir1_stops --slurpfile lines0 route_${route}_dir0_lines --slurpfile lines1 route_${route}_dir1_lines > route_${route}_feature

  #add to list of routes
  jq -c '.features += $feature' routes.json --slurpfile feature route_${route}_feature > routes.json.tmp
  mv routes.json.tmp routes.json

done

