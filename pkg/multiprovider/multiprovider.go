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
	"math"

	"k8s.io/client-go/kubernetes"
)

// Manager is the implementation to set up multitenancy.
type Manager struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
}

// NewManager returns a new multitenancy manager
func NewManager(kubeclientset kubernetes.Interface) *Manager {
	return &Manager{kubeclientset}
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
