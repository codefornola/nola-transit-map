#!/bin/bash

# Validate script arguments
if [ "$#" -ne 3 ]; then
    echo "Usage: $0 <input_targets.csv> <baseline_points.csv> <output.csv>" >&2
    exit 1
fi

INPUT_TARGETS="$1"
BASELINE_POINTS="$2"
OUTPUT_FILE="$3"

# Check if required files exist
if [ ! -f "$INPUT_TARGETS" ] || [ ! -f "$BASELINE_POINTS" ]; then
    echo "Error: One or both input CSV files do not exist." >&2
    exit 1
fi

# Write the header line for the new output CSV file
echo "shape_id,stop_lat,stop_lon" > "$OUTPUT_FILE"

# Process the input targets line-by-line, skipping the header
tail -n +2 "$INPUT_TARGETS" | while IFS=',' read -r -a columns; do
    # Extract latitude (5th column -> index 4) and longitude (6th column -> index 5)
    TARGET_LAT=$(echo "${columns[4]}" | tr -d '[:space:]')
    TARGET_LON=$(echo "${columns[5]}" | tr -d '[:space:]')

    # Skip rows that do not have valid numbers in the 5th and 6th slots
    if [[ ! "$TARGET_LAT" =~ ^-?[0-9.]+ ]] || [[ ! "$TARGET_LON" =~ ^-?[0-9.]+ ]]; then
        continue
    fi

    # Pass target coordinates into awk to query against the baseline file
    awk -F',' -v t_lat="$TARGET_LAT" -v t_lon="$TARGET_LON" '
    BEGIN {
        R = 6371 # Earth radius in kilometers
        min_dist = 99999999
        count = 0
        
        pi = atan2(0, -1)
        t_lat_rad = t_lat * pi / 180
        t_lon_rad = t_lon * pi / 180
    }
    # Read baseline file columns (assuming ID=col 1, Lat=col 2, Lon=col 3)
    NR > 1 && $2 ~ /^-?[0-9.]+/ && $3 ~ /^-?[0-9.]+/ {
        id = $1
        p_lat = $2
        p_lon = $3
        
        p_lat_rad = p_lat * pi / 180
        p_lon_rad = p_lon * pi / 180
        
        dlat = p_lat_rad - t_lat_rad
        dlon = p_lon_rad - t_lon_rad
        
        a = sin(dlat/2)^2 + cos(t_lat_rad) * cos(p_lat_rad) * sin(dlon/2)^2
        c = 2 * atan2(sqrt(a), sqrt(1-a))
        dist = R * c
        
        # Use a small delta check (1e-7) to account for floating-point precision ties
        if (dist < min_dist - 0.0000001) {
            min_dist = dist
            count = 1
            ids[count] = id
            lats[count] = p_lat
            lons[count] = p_lon
        } else if (awk_abs(dist - min_dist) < 0.0000001) {
            count++
            ids[count] = id
            lats[count] = p_lat
            lons[count] = p_lon
        }
    }
    function awk_abs(v) { return v < 0 ? -v : v }
    END {
        # Loop through all collected ties and print a row for each
        for (i = 1; i <= count; i++) {
            printf "%s,%s,%s\n", ids[i], t_lat, t_lon
        }
    }' "$BASELINE_POINTS" >> "$OUTPUT_FILE"

done
