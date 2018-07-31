### leader-election
- Leader election logic based on the [Bully Algorithm](https://en.wikipedia.org/wiki/Bully_algorithm) using grpc based communication.
- Each member boots up with a UUID as its identifier, which is exchanged between members.
- In the leader election mode, each member finds the greatest UUID using lexicographical comparison.
- Member having the greates UUID is elected as the leader.
- Before triggering the leader election mode, the flow of the algorithm makes sure that ll the members find and talk to each other to maintain
a consistent cluster state.
- Once leader is elected, it runs the leader work function and all other coordinating members stay in observer mode.
- Coordinating members ping the leader at every n externally configurable seconds to check, if the leader is alive and in a working state.
- when bad response if received from the leader, members will check one more time after a second before going into the ***re-election mode***
- Leader can also send tasks to the coordinators by defining more Service methods.