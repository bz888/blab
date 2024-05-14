module github.com/bz888/bad-siri/cmd

go 1.22.2

require (
	github.com/gdamore/tcell/v2 v2.7.1
	github.com/gordonklaus/portaudio v0.0.0-20230709114228-aafa478834f5
	github.com/rivo/tview v0.0.0-20240505185119-ed116790de0f
)

require (
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/icza/bitio v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mewkiz/flac v1.0.10 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sys v0.17.0 // indirect
	golang.org/x/term v0.17.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace github.com/bz888/bad-siri/api => ../api
replace github.com/bz888/bad-siri/speech => ../speech
