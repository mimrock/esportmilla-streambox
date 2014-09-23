esportmilla-streambox
=====================

The product is under development and not ready to use in production.

How to test
===========

Install go 1.2 or above.

Install dependencies with go get.

Add a MySQL database with a table called streambox_channels that contains two fields: string channel_id and int enabled. This table contains the list of the channel pool. Only the streams of these channels will be displayed.

`$ cp streambox.gcfg.default streambox.gcfg`

Edit streambox.gcfg

`$ go run streambox.go`

This should start a server that listens on the port given in the gcfg file. Visit localhost:8080/streambox to test the server. 

Query params:
=============

1. f: List of featured streams separated by a vertical bar. This determines the content of the top box.
2. g: List of games separated by a vertical bar. This determines the content of the bottom box.

Example:
--------

`http://localhost:8080/streambox?f=wingsofdeath&g=Hearthstone:+Heroes+of+Warcraft|League+of+Legends`

This query will show the stream of wingsofdeath it it's online, and the top five Hearthstone and LoL streams in the bottom box. If wingsofdeath's stream is not online, only the bottom box will be rendered, and it will be always visible.
