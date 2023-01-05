package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalVehicleTimestamp(t *testing.T) {
	testCases := []struct {
		vehicleJSON string
		wantTimeStr string
	}{
		{vehicleJSON: `{"tmstmp": "20200827 11:51"}`, wantTimeStr: "27 Aug 20 11:51 CDT"},
		{vehicleJSON: `{"tmstmp": "20230221 11:51"}`, wantTimeStr: "21 Feb 23 11:51 CST"},
	}

	for _, tc := range testCases {
		d := json.NewDecoder(strings.NewReader(tc.vehicleJSON))
		var gotVehicle Vehicle
		err := d.Decode(&gotVehicle)
		require.NoError(t, err)

		require.Equal(t, tc.wantTimeStr, gotVehicle.Tmstmp.Format(time.RFC822))
	}
}
