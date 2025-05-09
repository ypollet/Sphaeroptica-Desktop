module sphaeroptica.be/imports

require (
	github.com/h2non/bimg v1.1.9
	gonum.org/v1/gonum v0.15.1
	sphaeroptica.be/photogrammetry v1.0.0
)

replace sphaeroptica.be/photogrammetry v1.0.0 => ../photogrammetry

go 1.23.4
