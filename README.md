HN Notifications
============

Get an email as soon as a [Hacker News](https://news.ycombinator.com/) story reaches your custom score threshold.

[HN Notifications](http://hnnotifications.com) is a simple web service to fetch Hacker News items, and deliver email notification to its subscribers.
This small project has mainly been used to play around with Go and MongoDB, and although the current status is fully functional, there are a few things to polish up. So please, feel free to contribute!

## Getting started
You'll need Go (1.3+) and MongoDB

* Download & Install Go: [http://golang.org/doc/install](http://golang.org/doc/install)
* Install [MongoDB](http://docs.mongodb.org/manual/installation/)
* Some packages are managed with Mercurial or Bazaar. Ensure you have both `bzr` and `hg` installed in your path: [http://mercurial.selenic.com/](http://mercurial.selenic.com/), [http://wiki.bazaar.canonical.com/Download](http://wiki.bazaar.canonical.com/Download)
* Download and install dependencies: `go get & go build`
* Start MongoDB: `mongod [options]`
* Copy the sample config file `config.json.sample` into a new file `config.json`, under the same directory, and edit this file according to your system configuration (mongodb address, SMTP setup, etc)
* Run the app: `./hnnotifications`

The server will now be listening on the port specified on the config file (3000 by default): [http://localhost:3000/](http://localhost:3000/)

## License
This software is distributed under the BSD-style license found in the LICENSE file.