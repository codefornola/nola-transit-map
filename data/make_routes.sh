#!/bin/sh

#init list of routes
cat > routes.json << EOF
{
 "sources": [],
 "type": "FeatureCollection",
 "name": "routes",
 "crs": {
   "type": "name", 
   "properties": { "name": "urn:ogc:def:crs:OGC:1.3:CRS84" }
 },
 "features": []
}
EOF

add_features_from_GTFS () {
  url=$1
  need_stop_fixup=$2

  #get GTFS data
  wget -O GTFS.zip "$url"
  unzip -o GTFS.zip

  if $need_stop_fixup; then
    echo "FIXUP: determing shapeid for each stop point by finding nearest shape point"
    ./batch_distance.sh stops.txt shapes.txt stops_with_shapeid.txt
  fi

  routes=$(tail -n +2 routes.txt | cut -d "," -f 1)

  #add source of data
  cat feed_info.txt | sed 's/\r$//' |
    jq -c -R -s -f <(cat << EOF
      include "csv2json"; csv2json | .[0]
EOF
  ) > route_source
  jq -c '.sources += $source' routes.json --slurpfile source route_source > routes.json.tmp
  mv routes.json.tmp routes.json

  for route in $routes; do
    echo "ROUTE $route"

    #sometimes route_id is not the first column in trips.txt, so find its column number first
    route_id_column=$(head -1 trips.txt | tr ',' '\n' | grep -nx "route_id" | cut -d: -f1)

    #get geometry/stops for every shape in route
    shapeids=$(awk -F, -v col="$route_id_column" -v route="$route" '$col == route' trips.txt | cut -d ',' -f 8 | sort | uniq)

    shape_index=0
    for shape in $shapeids; do
      echo " SHAPE $shape_index: $shape"

      #convert shapes.txt lat/lons into geojson LineString
      (head -1 shapes.txt; grep "^$shape," shapes.txt) |
        jq -c -R -s 'include "csv2json"; csv2json | {type: "LineString", coordinates: [.[] | [.shape_pt_lon, .shape_pt_lat]]}' \
        > route_${route}_shape${shape_index}_lines

      #convert stops.txt lat/lons into geojson MutliPoint
      if $need_stop_fixup; then
          #NOTE: shapes and stops do not have idential lat/longs, so find stop in precalculated list of nearest shapeid
          (head -1 stops_with_shapeid.txt; grep "^$shape," stops_with_shapeid.txt) |
            jq -c -R -s 'include "csv2json"; csv2json | {type: "MultiPoint", coordinates: [.[] | [.stop_lon, .stop_lat]]}' \
            > route_${route}_shape${shape_index}_stops
      else
          #NOTE: since stops.txt doesnt have route info, correlate by matching route lat/lon with stops lat/lon
          (echo "stop_lat,stop_lon"; cat stops.txt | cut -d "," -f 5-6 | grep -f <(grep "^$shape," shapes.txt | cut -d "," -f 2-3)) |
            jq -c -R -s 'include "csv2json"; csv2json | {type: "MultiPoint", coordinates: [.[] | [.stop_lon, .stop_lat]]}' \
            > route_${route}_shape${shape_index}_stops
      fi
      shape_index=$(($shape_index+1))
    done

    #convert routes.txt entry to json route header
    (head -1 routes.txt; grep -e "^${route}," routes.txt ) |
      jq -c -R -s -f <(cat << EOF
        include "csv2json"; csv2json | .[0] |
          {
            type: "Feature",
            properties: { 
              route_id: .route_short_name,
              agency_id: .agency_id,
              route_short_name: .route_short_name,
              route_long_name: .route_long_name,
              route_type: .route_type,
              route_color: "\("#")\(.route_color)",
              route_text_color: "\("#")\(.route_text_color)",
            },
            geometry: { type: "GeometryCollection", geometries: [] }
          }
EOF
    ) > route_${route}_header

    #combine header/stops/lines into single geometrycollection for route
    filter="'.geometry.geometries += "
    spacer=""
    slurps=""
    shape_index=0
    for shape in $shapeids; do
      filter="${filter}${spacer}\$stops${shape_index} + \$lines${shape_index}"
      slurps="$slurps --slurpfile lines${shape_index} route_${route}_shape${shape_index}_lines"
      slurps="$slurps --slurpfile stops${shape_index} route_${route}_shape${shape_index}_stops"

      shape_index=$(($shape_index+1))
      spacer=" + "
    done
    filter="$filter'"
    cmd="jq $filter route_${route}_header "$slurps" > route_${route}_feature"
    bash -c "$cmd"

    #add to list of routes
    jq -c '.features += $feature' routes.json --slurpfile feature route_${route}_feature > routes.json.tmp
    mv routes.json.tmp routes.json

  done
}

echo "=== RTA ==="
add_features_from_GTFS https://www.norta.com/RTA/media/GTFS/GTFS.zip false

echo "=== JP TRANSIT ==="
#TODO: use proper URL when avail: https://rideneworleans.org/wp-content/uploads/JPT-20260720-GTFS.zip
add_features_from_GTFS http://localhost:8080/JPT-20260720-GTFS.zip true
