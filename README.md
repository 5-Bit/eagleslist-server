# eagleslist-server
The web backend for eagleslist

## How to build this code on your server

This code was build on ubutntu 14.04 and is not guanteed to run on another version of linux (but it should)

Step 1:
Install [golang](https://golang.org/dl/) on your server

Step 2: 
Install [postgresql](http://www.postgresql.org/download/) on your server, at version 9.3.9 or newer

Step 3:
Create a postgres user called appuser that is not a super user that has a password. 
Record the password somewhere you can find it.

Step 4:

Run `go get -u github.com/5-bit/eagleslist-server` to fetch this project and all of it's dependencies

Step 5:
Run the SQL code in schema.sql as the postgres user

Step 6:
Run go build
Step 7:
Run `sudo setcap 'cap_net_service=eq' ./eaglelist` to allow eaglelist bind to low ports, like 80

Step 8: 

Run `nohup ./eaglelist &` to start the server.

Fin.
