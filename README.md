# outboxen

Outboxen is a library for implementing the [transactional outbox pattern][transactional-outbox-pattern] in Go.

I found that there weren't many libraries for implementing this in Go, and the ones that did exist made different design
trade-offs than I would make.

For an example using MySQL via [GORM][gorm], see the [outboxen-gorm][outboxen-gorm]
example [here][outboxen-gorm-example].

## Features

* Makes no assumptions about your storage mechanism, you provide:
    * A way to atomically claim entries in storage for a given processor
    * A way to retrieve entries claimed by a given processor
    * A way to delete entries
* Compatible with horizontal scaling
    * Safely claims outbox entries for publishing, with a deadline for gracefully tolerating failures
* Designed not to interfere with the _transactional_ part of "transactional outbox pattern"
    * It does not create transactions for you
    * You write entries to your outbox storage in the same transaction as your state modification

## Drivers

* [outboxen-gorm][outboxen-gorm] - implements the storage layer using [GORM][gorm]

[transactional-outbox-pattern]: https://microservices.io/patterns/data/transactional-outbox.html

[outboxen-gorm]: https://github.com/omaskery/outboxen-gorm

[gorm]: https://gorm.io/

[outboxen-gorm-example]: https://github.com/omaskery/outboxen-gorm/tree/main/examples/mysql
