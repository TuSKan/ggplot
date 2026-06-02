// Package main provides a reusable 3D attractor engine and beautiful presets.
//
// The engine integrates ODE systems using 4th-order Runge-Kutta (RK4) and
// includes presets for Lorenz, Rössler, Halvorsen, Thomas, and Chen attractors.
package main

import "math"

// Vec3 is a 3D point/vector.
type Vec3 struct {
	X float64
	Y float64
	Z float64
}

// Flow3D computes the derivative dv/dt at a given state v.
type Flow3D func(v Vec3) Vec3

// Attractor3DParams configures the attractor integration.
type Attractor3DParams struct {
	Steps  int
	BurnIn int
	Dt     float64
	Start  Vec3
	Flow   Flow3D
}

// Attractor3DData holds the integrated trajectory.
type Attractor3DData struct {
	X []float64
	Y []float64
	Z []float64

	// T is useful for color gradients (normalized [0, 1]).
	T []float64
}

// Attractor3D integrates a 3D ODE system using RK4.
func Attractor3D(p Attractor3DParams) Attractor3DData {
	if p.Steps <= 0 {
		return Attractor3DData{}
	}
	if p.BurnIn < 0 {
		p.BurnIn = 0
	}
	if p.Dt == 0 {
		p.Dt = 0.005
	}
	if p.Flow == nil {
		return Attractor3DData{}
	}

	xs := make([]float64, p.Steps)
	ys := make([]float64, p.Steps)
	zs := make([]float64, p.Steps)
	ts := make([]float64, p.Steps)

	v := p.Start
	total := p.Steps + p.BurnIn
	j := 0

	for i := range total {
		v = rk4(v, p.Dt, p.Flow)

		if i >= p.BurnIn {
			xs[j] = v.X
			ys[j] = v.Y
			zs[j] = v.Z
			ts[j] = float64(j) / float64(max(1, p.Steps-1))
			j++
		}
	}

	return Attractor3DData{
		X: xs,
		Y: ys,
		Z: zs,
		T: ts,
	}
}

func rk4(v Vec3, dt float64, f Flow3D) Vec3 {
	k1 := f(v)
	k2 := f(add(v, scale(k1, dt*0.5)))
	k3 := f(add(v, scale(k2, dt*0.5)))
	k4 := f(add(v, scale(k3, dt)))

	return Vec3{
		X: v.X + dt*(k1.X+2*k2.X+2*k3.X+k4.X)/6, //nolint:mnd // RK4 weights are part of the algorithm.
		Y: v.Y + dt*(k1.Y+2*k2.Y+2*k3.Y+k4.Y)/6, //nolint:mnd // RK4 weights are part of the algorithm.
		Z: v.Z + dt*(k1.Z+2*k2.Z+2*k3.Z+k4.Z)/6, //nolint:mnd // RK4 weights are part of the algorithm.
	}
}

func add(a, b Vec3) Vec3 {
	return Vec3{
		X: a.X + b.X,
		Y: a.Y + b.Y,
		Z: a.Z + b.Z,
	}
}

func scale(v Vec3, s float64) Vec3 {
	return Vec3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

// ---------------------------------------------------------------------------
// Attractor presets
// ---------------------------------------------------------------------------

// Lorenz returns the Lorenz system flow.
// Classic parameters: sigma=10, rho=28, beta=8/3.
func Lorenz(sigma, rho, beta float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: sigma * (v.Y - v.X),
			Y: v.X*(rho-v.Z) - v.Y,
			Z: v.X*v.Y - beta*v.Z,
		}
	}
}

// LorenzBeautiful generates an n-point Lorenz attractor with tuned parameters.
func LorenzBeautiful(n int) Attractor3DData {
	return Attractor3D(Attractor3DParams{
		Steps:  n,
		BurnIn: 2_000,
		Dt:     0.005,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Lorenz(10, 28, 8.0/3.0),
	})
}

// Rossler returns the Rössler system flow.
// Classic parameters: a=0.2, b=0.2, c=5.7.
func Rossler(a, b, c float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: -v.Y - v.Z,
			Y: v.X + a*v.Y,
			Z: b + v.Z*(v.X-c),
		}
	}
}

// RosslerBeautiful generates an n-point Rössler attractor with tuned parameters.
func RosslerBeautiful(n int) Attractor3DData {
	return Attractor3D(Attractor3DParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.01,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Rossler(0.2, 0.2, 5.7),
	})
}

// Halvorsen returns the Halvorsen symmetric attractor flow.
// a=1.3 produces the chaotic tri-lobed attractor; a=1.89 converges to a periodic orbit.
func Halvorsen(a float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: -a*v.X - 4*v.Y - 4*v.Z - v.Y*v.Y,
			Y: -a*v.Y - 4*v.Z - 4*v.X - v.Z*v.Z,
			Z: -a*v.Z - 4*v.X - 4*v.Y - v.X*v.X,
		}
	}
}

// HalvorsenBeautiful generates an n-point Halvorsen attractor with tuned parameters.
func HalvorsenBeautiful(n int) Attractor3DData {
	return Attractor3D(Attractor3DParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.005,
		Start:  Vec3{X: 1, Y: 0, Z: 0},
		Flow:   Halvorsen(1.3),
	})
}

// Thomas returns the Thomas cyclically symmetric attractor flow.
// Classic parameter: b=0.208186.
func Thomas(b float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: math.Sin(v.Y) - b*v.X,
			Y: math.Sin(v.Z) - b*v.Y,
			Z: math.Sin(v.X) - b*v.Z,
		}
	}
}

// ThomasBeautiful generates an n-point Thomas attractor with tuned parameters.
func ThomasBeautiful(n int) Attractor3DData {
	return Attractor3D(Attractor3DParams{
		Steps:  n,
		BurnIn: 10_000,
		Dt:     0.05,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Thomas(0.208186),
	})
}

// Chen returns the Chen system flow.
// Classic parameters: a=35, b=3, c=28.
func Chen(a, b, c float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: a * (v.Y - v.X),
			Y: (c-a)*v.X - v.X*v.Z + c*v.Y,
			Z: v.X*v.Y - b*v.Z,
		}
	}
}

// ChenBeautiful generates an n-point Chen attractor with tuned parameters.
func ChenBeautiful(n int) Attractor3DData {
	return Attractor3D(Attractor3DParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.0025,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Chen(35, 3, 28),
	})
}
