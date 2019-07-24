# go-nordlead3
Nord Lead 3 Sysex Parser Library in Go (Golang). Work in Progress.

## Kicking the tires

For a quick spin through it, grab a sysex file or saved dump from your NL3 and run:

`go run nl3edit.go <path to your sysex file>`

If you want to get fancy, append ` p <X> <Y> <D>` to that to print out the program or performance in bank X location Y, whichever isn't blank. If the sysex contains data for that location, it'll print the actual parameter values for the patch in question.

Where:
* X is the program or performance bank you want to view
* Y is the location in that bank
* D is the depth of data structure to print

