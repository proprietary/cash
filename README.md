**This is a fork of [github.com/zelcon5/cash](https://github.com/zelcon5/cash) with added support for negatives.**

## Cash

A realistic money library for Go aka Golang. There is no BigDecimal equivalent for Go, so I found myself using integral cents for money. This library offers convenience by making "money" or "cash" a real type.

Money is internally stored as cents (`int64` string).

Heavy inspiration from Martin Fowler's "Quantity" object
