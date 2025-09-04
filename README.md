# exam-monitor

### To run on windows, download executable and double click
### Use terminal to run on linux
- **windows**: [examgaurd-student-v1.0.0](https://github.com/khayrultw/exam-monitor/releases/download/v1.0.0-rc/examgaurd_student.exe)
- **windows**: [examgaurd-teacher-v1.0.0](https://github.com/khayrultw/exam-monitor/releases/download/v1.0.0-rc/examgaurd_teacher)

## To build 

#### Make sure you have golang installed

##### To run the server, navigate to server-go and execute on linux
Install following packages
`libvulkan-dev libxkbcommon-x11-dev libx11-xcb-dev`
Then
`go build . && ./server`

#### To build executable for windows, navigate to client-go and execute
`x86_64-w64-mingw32-windres app.rc -O coff -o app-res.o`
<br>
`GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -ldflags "-H=windowsgui -extldflags=-Wl,app-res.o" -o examgaurd.exe .`

##### To run the client, navigate to client-go then open terminal and execute the below command
`go build . && ./client`

### For Mac
#### First clone the repo, navigate to client and run below commands
`brew install go`
<br>
`go mod tidy`
<br>
`go run main.go`

