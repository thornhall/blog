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
Our data model is simple, it includes only **buy orders** and **sell orders**.