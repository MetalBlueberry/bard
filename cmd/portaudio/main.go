package main

import "github.com/gordonklaus/portaudio"

func main() {

	err := portaudio.Initialize()
	if err != nil {
		panic(err)
	}
	in, err := portaudio.DefaultInputDevice()
	if err != nil {
		panic(err)
	}

	in := make([]int32, 64)
	stream, err := portaudio.OpenDefaultStream(1, 0, 44100, len(in), in)
	if err != nil {
		panic(err)
	}
	defer stream.Close()

	err = stream.Start()
	if err != nil {
		panic(err)
	}

	stream.Read()

	stream.

}
