#!/bin/sh

#init list of routes
cat > routes.json << EOF
{
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

  #get GTFS data
  wget -O GTFS.zip "$url"
  unzip -o GTFS.zip

  routes=$(tail -n +2 routes.txt | cut -d "," -f 1)

  #routes.txt lists as '53-O', but shapes.txt has it as just 53. fix it here
  routes=${routes/53-O/53}

  for route in $routes; do
    echo "ROUTE $route"

    #get geometry/stops for every direction
    dirs=$(grep "\-${route}-" shapes.txt | cut -d ',' -f 1 | sort | uniq | cut -d '-' -f 3)
    dir_index=0
    for dir in $dirs; do
      echo " DIR $dir_index: $dir"

      #convert shapes.txt lat/lons into geojson LineString
      (head -1 shapes.txt; grep "shp-$route-$dir" shapes.txt) |
        jq -c -R -s 'include "csv2json"; csv2json | {type: "LineString", coordinates: [.[] | [.shape_pt_lon, .shape_pt_lat]]}' \
        > route_${route}_dir${dir_index}_lines

      #convert stops.txt lat/lons into geojson MutliPoint
      #NOTE: since stops.txt doesnt have route info, correlate by matching route lat/lon with stops lat/lon
      (echo "stop_lat,stop_lon"; cat stops.txt | cut -d "," -f 5-6 | grep -f <(grep "shp-$route-$dir" shapes.txt | cut -d "," -f 2-3)) |
        jq -c -R -s 'include "csv2json"; csv2json | {type: "MultiPoint", coordinates: [.[] | [.stop_lon, .stop_lat]]}' \
        > route_${route}_dir${dir_index}_stops

      dir_index=$(($dir_index+1))
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
    dir_index=0
    for dir in $dirs; do
      filter="${filter}${spacer}\$stops${dir_index} + \$lines${dir_index}"
      slurps="$slurps --slurpfile lines${dir_index} route_${route}_dir${dir_index}_lines"
      slurps="$slurps --slurpfile stops${dir_index} route_${route}_dir${dir_index}_stops"

      dir_index=$(($dir_index+1))
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
add_features_from_GTFS https://www.norta.com/RTA/media/GTFS/GTFS.zip

echo "=== JP TRANSIT ==="
add_features_from_GTFS https://rideneworleans.org/wp/wp-content/uploads/GTFS-JET-20240913.zip
