package geom

// Geometry compiles data and aesthetic mappings into generic drawing parameters.
type Geometry interface {
	// Compile produces abstract draw commands.
	Compile() error
}

// Point geometry implementation.
type Point struct{}

// Line geometry implementation.
type Line struct{}

// Bar geometry implementation.
type Bar struct{}

// Polygon geometry implementation.
type Polygon struct{}
