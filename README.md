# Postman

A tool to send emails in batch. It reads recipients from a .csv file and
writes status to the same file. The status will be used when resending.

### How to build for Windows? ###
```
GOOS=windows GOARCH=386 go build -o postman.exe main.go
```
