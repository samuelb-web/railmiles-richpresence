mkdir -p out

GOOS=linux GOARCH=amd64  go build -o -ldflags -H=windowsgui ./out/railmiles-richpresence.out
#GOOS=darwin GOARCH=amd64 go build -o ./out/railmiles-richpresence.mac_intel.out
#GOOS=darwin GOARCH=arm64 go build -o ./out/railmiles-richpresence.mac_m1.out
GOOS=windows GOARCH=amd64 go build -o -ldflags -H=windowsgui ./out/railmiles-richpresence.exe
