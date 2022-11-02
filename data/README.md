# routes

Derived from GTFS trip data.

We used the "dissolve" process in QGIS to combine all trip data per route id. This still leaves a lot of unnecessary segments at bus stops.

One way to improve the geometry of the routes is replace the feature geometries with centerlines from this dataset in data.nola.gov ([Centerline](https://data.nola.gov/Transportation-and-Infrastructure/Centerline/hp2r-gr3h)). I did this in QGIS for the General Meyer Local route (103) just to try it out, and these are the steps I used:

1. Select each centerline segment that lies along a single route (segments are split at all intersections)
2. Export these selected segments to a new layer
3. Touch up vertices as needed and then dissolve the layer to a single feature
4. Copy this dissolved feature into the main `routes.geojson` file
5. Copy over attributes from original route feature
6. Delete the original feature for that route.
7. Leave an comment in the edit_comment field

Leaving these notes here for future reference, as the features can be slowly fixed up over time.