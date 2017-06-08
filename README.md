# Logs Monitor

The tool for logs monitoring. 

Compile and drop it on a server behind a proxy and auth. Login and add as many files as you want to monitor them on the one screen at the same time in almost realtime.

The repository consists of:

* a package `logmon` with basic interface for watching files like `tail -f` program
* a server which consumes the package and exposes a REST API using websockets to push incoming data in almost realtime (1 second delay)
* an example JavaScript vuejs frontend application which consumes the REST API using sockets and exposes a user-friendly graphical interface for managing and watching logs' flow

