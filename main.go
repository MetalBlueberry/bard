package main

import (
	"github.com/andrepxx/go-dsp-guitar/controller"
	"github.com/andrepxx/go-dsp-guitar/hwio"
	"github.com/andrepxx/go-dsp-guitar/path"
	"github.com/andrepxx/go-dsp-guitar/tuner"
	"github.com/andrepxx/go-dsp-guitar/wave"
)

func main() {
	tn := tuner.Create()
	tn.Analyze()
	// tn.Process()
	ctrl := controller.CreateController()
	ctrl.Operate(1)
}
