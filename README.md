HN Notifications
============

Get an email as soon as a [Hacker News](https://news.ycombinator.com/) story reaches your custom score threshold.

[HN Notifications](http://hnnotifications.com) is a simple web service to fetch Hacker News items, and deliver email notification to its subscribers.
This small project has mainly been used to play around with Go and MongoDB, and although the current status is fully functional, there are a few things to polish up. So please, feel free to contribute!

## How it works

In a nutshell, the notification process runs every 15 minutes, getting the top 100 HN items (via the official [Firebase API](https://github.com/HackerNews/API)), and delivers an email to the corresponding users subscribed to the service. This implementation does **not** make use of the *Live Data* feature of Firebase, which would in turn be more efficient.

Authentication mechanism is currently minimalist: any configuration in the subscription settings is confirmed through a verification email. Therefore no username or password is required.

## Getting started
You'll need Go (1.3+) and MongoDB:

* Install [Go]([http://golang.org/doc/install](http://golang.org/doc/install)) and [MongoDB](http://docs.mongodb.org/manual/installation/).
* Some packages are managed with Mercurial or Bazaar. Ensure you have both `bzr` and `hg` installed in your path: [http://mercurial.selenic.com/](http://mercurial.selenic.com/), [http://wiki.bazaar.canonical.com/Download](http://wiki.bazaar.canonical.com/Download).
* Install dependencies, and build the app: `go get & go build`.
* Start MongoDB: `mongod [options]`.
* Copy the sample config file `config.json.sample` into a new file `config.json`, under the same directory, and edit this file according to your system configuration (mongodb address and credentials, SMTP setup, etc).
* Run the app: `./hnnotifications`.

The server will now be listening on the port specified in the config file (3000 by default): [http://localhost:3000/](http://localhost:3000/).

## License
This software is distributed under the BSD-style license found in the LICENSE file.