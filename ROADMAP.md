# Gochan Development Roadmap

This is just a very preliminary roadmap to map out what I plan to focus on with the versions


1.x
----
* Improve posting stability (almost done)
* Add management functions to make things simpler
* Fix bugfixes (almost done, as far as we can tell)
* Add some kind of database schema to handle any possible changes in the database structure with new versions (if there are any)
* Add functionality to aid new admins in transferring boards from other imageboard systems (TinyBoard, Kusaba, etc) to Gochan. This would largely be a database issue (see above)

2.x
----
* Add functionality to go above and beyond what you would expect from an imageboard, without worrying about feature creep. Go's speed would allow us to do this without causing Gochan to slow down
* Add a manage function to modify the configuration without having to directly modify gochan.json directly
* Add a plugin system, likely using Lua to make it lightweight but still powerful. Go's package system would make this pretty easy and straightforward
* Add a mange function to make it easier to update gochan. Not just the database schema but with the system in general.

3.x
----
* ???
* I'll figure this out when we get to 2.x but feel free to put suggestions in the [issue page](https://github.com/Eggbertx/gochan/issues), or make a post at http://gochan.org/
