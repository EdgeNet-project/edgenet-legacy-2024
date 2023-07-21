/*
Copyright 2022 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package multiprovider

import (
	"fmt"
	"log"
	"math"
	"strconv"

	clientset "github.com/EdgeNet-project/edgenet/pkg/generated/clientset/versioned"

	"k8s.io/client-go/kubernetes"
)

// Manager is the implementation to set up multitenancy.
type Manager struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// edgeclientset is a clientset for the EdgeNet API groups
	edgeclientset clientset.Interface
	// remotekubeclientset is a standard kubernetes clientset for remote clusters
	remotekubeclientset kubernetes.Interface
	// remoteedgeclientset is a clientset for the EdgeNet API groups for remote clusters
	remoteedgeclientset clientset.Interface
}

// NewManager returns a new multitenancy manager
func NewManager(kubeclientset, remotekubeclientset kubernetes.Interface, edgeclientset, remoteedgeclientset clientset.Interface) *Manager {
	return &Manager{kubeclientset, edgeclientset, remotekubeclientset, remoteedgeclientset}
}

// GeoFence function determines whether the point is inside a polygon by using the crossing number method.
// This method counts the number of times a ray starting at a point crosses a polygon boundary edge.
// The even numbers mean the point is outside and the odd ones mean the point is inside.
func GeoFence(boundbox []float64, polygon [][]float64, x float64, y float64) bool {
	vertices := len(polygon)
	lastIndex := vertices - 1
	oddNodes := false
	if boundbox[0] <= x && boundbox[1] >= x && boundbox[2] <= y && boundbox[3] >= y {
		for index := range polygon {
			if (polygon[index][1] < y && polygon[lastIndex][1] >= y || polygon[lastIndex][1] < y &&
				polygon[index][1] >= y) && (polygon[index][0] <= x || polygon[lastIndex][0] <= x) {
				if polygon[index][0]+(y-polygon[index][1])/(polygon[lastIndex][1]-polygon[index][1])*
					(polygon[lastIndex][0]-polygon[index][0]) < x {
					oddNodes = !oddNodes
				}
			}
			lastIndex = index
		}
	}
	return oddNodes
}

// Boundbox returns a rectangle which created according to the points of the polygon given
func Boundbox(points [][]float64) []float64 {
	var minX float64 = math.MaxFloat64
	var maxX float64 = -math.MaxFloat64
	var minY float64 = math.MaxFloat64
	var maxY float64 = -math.MaxFloat64

	for _, coordinates := range points {
		minX = math.Min(minX, coordinates[0])
		maxX = math.Max(maxX, coordinates[0])
		minY = math.Min(minY, coordinates[1])
		maxY = math.Max(maxY, coordinates[1])
	}

	bounding := []float64{minX, maxX, minY, maxY}
	return bounding
}

// GetGeolocationLabelsByIP returns geolabels from the MaxMind GeoIP2 precision service
func (m *Manager) GetGeolocationLabelsByIP(
	maxmindURL string,
	maxmindAccountID string,
	maxmindLicenseKey string,
	address string,
	patch bool,
) (map[string]string, bool) {
	// Fetch geolocation information
	record, err := getMaxmindLocation(maxmindURL, maxmindAccountID, maxmindLicenseKey, address)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	continent := sanitizeNodeLabel(record.Continent.Names["en"])
	country := record.Country.IsoCode
	state := record.Country.IsoCode
	city := sanitizeNodeLabel(record.City.Names["en"])
	isp := sanitizeNodeLabel(record.Traits.Isp)
	as := sanitizeNodeLabel(record.Traits.AutonomousSystemOrganization)
	asn := strconv.Itoa(record.Traits.AutonomousSystemNumber)
	var lon string
	var lat string
	if record.Location.Longitude >= 0 {
		lon = fmt.Sprintf("e%.6f", record.Location.Longitude)
	} else {
		lon = fmt.Sprintf("w%.6f", record.Location.Longitude)
	}
	if record.Location.Latitude >= 0 {
		lat = fmt.Sprintf("n%.6f", record.Location.Latitude)
	} else {
		lat = fmt.Sprintf("s%.6f", record.Location.Latitude)
	}
	if len(record.Subdivisions) > 0 {
		state = record.Subdivisions[0].IsoCode
	}

	// Create label map to attach to the node
	keyPrefix := "edge-net.io/"
	if patch {
		keyPrefix = "edge-net.io~1"
	}
	geoLabels := map[string]string{
		fmt.Sprintf("%s%s", keyPrefix, "continent"):   continent,
		fmt.Sprintf("%s%s", keyPrefix, "country-iso"): country,
		fmt.Sprintf("%s%s", keyPrefix, "state-iso"):   state,
		fmt.Sprintf("%s%s", keyPrefix, "city"):        city,
		fmt.Sprintf("%s%s", keyPrefix, "lon"):         lon,
		fmt.Sprintf("%s%s", keyPrefix, "lat"):         lat,
		fmt.Sprintf("%s%s", keyPrefix, "isp"):         isp,
		fmt.Sprintf("%s%s", keyPrefix, "as"):          as,
		fmt.Sprintf("%s%s", keyPrefix, "asn"):         asn,
	}
	return geoLabels, true
}
