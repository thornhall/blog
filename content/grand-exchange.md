---
title: How I Created a Grand Exchange Clone with Go and PostgreSQL
slug: grand-exchange
date: Nov 20, 2025
category: Engineering
excerpt: Trading with other players at scale.
---
To understand the Grand Exchange, we need to understand trading in video games. Say player 1
mines 100 iron ore and wishes to sell it. Player 2 needs 100 iron ore to make iron bars for armor they want to make.
Player 1 can _trade_ directly with player 2 the 100 iron ore in exchange for some gold.

## The Grand Exchange
The GE solves this problem at scale. Now, instead of trading directly with player 2, I can put a **sell order** into the GE.
Then, the GE will automatically match the sell order with other players who are buying via their own buy orders.

## The Architecture
Our application will be a REST API written in Go. The Go servers will contain no user-specific data in-memory, meaning we can scale it
horizontally as needed. For the database, we will use PostgreSQL, a highly scalable relational database. We're choosing a relational database
for a few reasons:

- We need to make sure each order is consumed only once.
- Using database transactions, we can ensure that all operations on some data either all succeed or all fail, ensuring data integrity.
- A relational database allows us to scan for related data quickly. Some order matching logic will rely on data relations by JOINing tables together.

## Our Data Model
Our data model is simple, it includes only **buy orders** and **sell orders**. It looks something like this:

```go
package model

type OrderStatus string

const (
	StatusActive    OrderStatus = "active"
	StatusFilled    OrderStatus = "filled"
	StatusPartial   OrderStatus = "partial"
	StatusCancelled OrderStatus = "cancelled"
)

type Order struct {
	ID                int64    
	ItemID            string   
	TotalQuantity     int64      
	RemainingQuantity int64     
	Status            OrderStatus 
}

type BuyOrder struct {
	Order
	BidPrice   int64 
	BuyerID    int64 
	TotalSpent int64 
}

type SellOrder struct {
	Order
	AskPrice int64
	SellerID int64 
}
```

Our database table is almost an exact reflection of the above model. However, there's one important detail missing: the indexes.

## What is a Database Index?
Database indexes speed up queries to the database by "pre-packaging" the data in a specific way. With careful planning, we can optimize our database
to handle many orders of magnitude of extra traffic that we wouldn't be able to without the indexes.

For our Sell Orders table, the indexes look like this:
```sql
CREATE INDEX idx_item_id_ask_price ON sell_orders (item_id, ask_price ASC);
CREATE INDEX idx_seller_id ON sell_orders (seller_id);
```

Why did I choose these indexes? The answer is I examined what queries my application will use and based the indexes off of that.

For example, when a buy order is created, to find matching sell orders we do the following query:
```sql
SELECT id, total_quantity, remaining_quantity, offer_status, ask_price
FROM sell_orders
WHERE item_id = $1
AND ask_price <= $2
AND offer_status IN ('active', 'partial')
AND seller_id != $3
ORDER BY ask_price ASC
LIMIT 10
FOR UPDATE SKIP LOCKED;
```

This index optimizes that query:
```sql
CREATE INDEX idx_item_id_ask_price ON sell_orders (item_id, ask_price ASC);
```

This is because the query is looking for orders with a specific `item_id`. It also checks the ask price. It just so happens that b-tree indexes
leave the data in sorted order, which is perfect for how we're querying on `ask_price`.

This index:
```sql
CREATE INDEX idx_seller_id ON sell_orders (seller_id);
```
was created in intention of selecting all of a user's active orders to show them in some dashboard.

The query:

```sql
SELECT id, total_quantity, remaining_quantity, offer_status, ask_price
FROM sell_orders
WHERE seller_id = $1
AND offer_status IN ('active', 'partial')
LIMIT 10
```

Let's examine the query for matching Buy Orders to Sell Orders closer to understand some key properties of the system.

```sql
SELECT id, total_quantity, remaining_quantity, offer_status, ask_price
FROM sell_orders
WHERE item_id = $1
AND ask_price <= $2
AND offer_status IN ('active', 'partial')
AND seller_id != $3
ORDER BY ask_price ASC
LIMIT 10
FOR UPDATE SKIP LOCKED;
```
This query is saying:
> Get me all the sell orders whose item ID and ask price match the buy order's item ID and bid price. Limit to 10 orders. Lock these orders and ignore locked orders.

Why do we need to lock orders? Well, we need to ensure that an order is consumed exactly once. If we didn't, it would be possible for an order
to be consumed past its TotalQuantity by another player, defeating the purpose of the trading system.

There's two problems though. Since we're locking the orders, what happens if two orders come in at the same time competing for the same 10 sell orders,
and there are only 10 sell orders in the DB? The first player to lock the 10 orders will lock all 10, starving the other player of orders in the case
that the first player doesn't actually _need_ all 10 orders.

The second problem occurs if the 10 orders that the first player locked do not completely fulfill the first player's order. 
How do we fill the order the rest of the way, assuming there are still extra orders in the DB?

## The Reconciliation Worker

To solve these problems, we will spawn a worker in a goroutine that:
- Periodically wakes up and checks the database for active or partial orders which are not locked currently
- Matches those orders with the opposite order type

For simplicity, the reconciliation worker will run alongside the REST API. If we reach a massive scale for our game, we may need to make the
reconciliation worker its own application so that we may scale it independently.

## App Structure
Our app is structured like a typical Go app. Implementation details are in `internal`, and the entrypoint of our app is in `cmd/api/main.go`.
If you've read my other articles, the patterns might be similar. The hierarchy looks like this:

```go
// Server -> Router -> Handler -> Repo -> DB
```

Here, we have a server listening for internet traffic. When the server receives a request, it looks to the router to figure out which handler
to execute. Then, the corresponding handler will execute the database logic.

## Testing
How do we test our app, given that we need a Postgres database for our app to run? The answer is the `testcontainers` library. This library
gives our code access to a Docker container running a Postgres instance, conveniently all from our Go code. The major downside to this approach
is that it introduces quite a few dependencies into our app. An alternative approach could be to run our own docker container.

## Conclusion
That's the overview of the grand exchange app. In the future, I plan to prototype it to use for a video game I am making called Path of Magic.