gorm-cursor-paginator
[![Build Status](https://travis-ci.org/pilagod/gorm-cursor-paginator.svg?branch=master)](https://travis-ci.org/pilagod/gorm-cursor-paginator)
[![Coverage Status](https://coveralls.io/repos/github/pilagod/gorm-cursor-paginator/badge.svg?branch=master)](https://coveralls.io/github/pilagod/gorm-cursor-paginator?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/pilagod/gorm-cursor-paginator)](https://goreportcard.com/report/github.com/pilagod/gorm-cursor-paginator)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/6d8f88386eeb401b8804bb78f372b346)](https://app.codacy.com/app/pilagod/gorm-cursor-paginator?utm_source=github.com&utm_medium=referral&utm_content=pilagod/gorm-cursor-paginator&utm_campaign=Badge_Grade_Dashboard)
=====================

A paginator doing cursor-based pagination based on [GORM](https://github.com/jinzhu/gorm)

Installation
------------

```sh
go get -u github.com/pilagod/gorm-cursor-paginator
```

Usage by Example
----------------

Assume there is an query struct for paging:

```go
type PagingQuery struct {
    After  *string
    Before *string
    Limit  *int
    Order  *string
}
```

and a GORM model:

```go
type Model struct {
    ID          int
    CreatedAt   time.Time
}
```

You can simply build up a new cursor paginator from the PagingQuery like:

```go
import (
    paginator "github.com/pilagod/gorm-cursor-paginator"
)

func GetModelPaginator(q PagingQuery) *paginator.Paginator {
    p := paginator.New()

    p.SetKeys("CreatedAt", "ID") // [default: "ID"] (supporting multiple keys, order of keys matters)

    if q.After != nil {
        p.SetAfterCursor(*q.After) // [default: nil]
    }

    if q.Before != nil {
        p.SetBeforeCursor(*q.Before) // [default: nil]
    }

    if q.Limit != nil {
        p.SetLimit(*q.Limit) // [default: 10]
    }

    if q.Order != nil && *q.Order == "asc" {
        p.SetOrder(paginator.ASC) // [default: paginator.DESC]
    }
    return p
}
```

Then you can start to do pagination easily with GORM:

```go
func Find(db *gorm.DB, q PagingQuery) ([]Model, paginator.Cursor, error) {
    var models []Model

    stmt := db.Where(/* ... other filters ... */)
    stmt = db.Or(/* ... more other filters ... */)

    // get paginator for Model
    p := GetModelPaginator(q)

    // use GORM-like syntax to do pagination
    result := p.Paginate(stmt, &models)

    if result.Error != nil {
        // ...
    }
    // get cursor for next iteration
    cursor := p.GetNextCursor()

    return models, cursor, nil
}
```

After paginating, you can call `GetNextCursor()`, which returns a `Cursor` struct containing cursor for next iteration:

```go
type Cursor struct {
    After  *string `json:"after"`
    Before *string `json:"before"`
}
```

That's all ! Enjoy your paging in the GORM world :tada:

License
-------

© Chun-Yan Ho (pilagod), 2018-NOW

Released under the [MIT License](https://github.com/pilagod/gorm-cursor-paginator/blob/master/LICENSE)
