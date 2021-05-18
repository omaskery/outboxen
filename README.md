# outboxen

Outboxen is a library for implementing the [transactional outbox pattern](transactional-outbox-pattern) in Go.

I found that there weren't many libraries for implementing this in Go, and the ones that did exist made
different design trade-offs than I would make.

## Features

* Makes no assumptions about your storage mechanism, you provide:
  * A way to atomically claim entries in storage for a given processor
  * A way to retrieve entries claimed by a given processor
  * A way to delete entries
* Compatible with horizontal scalability
  * Safely claims outbox entries for publishing, with a deadline for gracefully tolerating failures
* Designed not to interfere with the _transactional_ part of "transactional outbox pattern"
  * It does not create transactions for you 
  * You write entries to your outbox storage in the same transaction as your state modification

[transactional-outbox-pattern]: https://microservices.io/patterns/data/transactional-outbox.html
