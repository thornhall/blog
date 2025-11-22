---
title: Improving the Security of the Blog
slug: improving-durability
date: Nov 22, 2025
category: Engineering
excerpt: A lesson learned.
---
I made a mistake. 

When making this blog, I trusted that the internet would treat me nicely and not attempt to break it. After all,
it's just a friendly blog, right?

Well, I was wrong. What mistake did I make?

To understand the issue, we need to understand how likes and views worked before the security updates.

## The Mistake
When designing likes and views, I decided the following:
- Likes and Views would be incremented via a public endpoint.
- To prevent one single user from amassing many views and likes, I cached data in the client browser's local storage.

This meant that the only thing preventing a user from spamming the `/likes` endpoint hundreds, or even thousands of times was
via checking the timestamp on the data stored in local storage.

It also meant a user would have to go out of their way to write a malicious script to fraudulently hit the `/likes` endpoint.

I knew this vulnerability existed when I published the first version of the blog, but I thought - surely the internet would get nothing
out of trying to ruin my harmless blog. So for the MVP, I didn't attempt to dedupe likes and views at the API level. This was a mistake.

## What the 'Bad Actor' Did
The bad actor simply inflated the likes count of one of my posts to be greater than the views count, surely in an attempt to try to make me look bad. 
Thankfully, I was able to resolve the issue within minutes of noticing. That's what I get for having faith in the internet, I suppose. 

## How I Fixed it
Let's dive deeper into the more important details of the article: how did I fix this vulnerability? I'll also discuss limitations of that fix.

I decided to identify users at the IP level. Nothing fancy - but when a user "likes" or "views" a post, we'll simply record that their IP address
has done so in the past, and any subsequent requests with the same IP will fail.

## Changes at the Database Level
To implement this, we create two new tables, `ip_likes` and `ip_views`.

Our repository logic for incrementing views, as an example, looks like this after the update:

```go
func (r *Repo) IncrementViews(ctx context.Context, ip, slug string) (Stats, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Stats{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
        INSERT INTO ip_views (ip, post_slug) 
        VALUES (?, ?) 
        ON CONFLICT(ip, post_slug) DO NOTHING;
    `, ip, slug)
	if err != nil {
		return Stats{}, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return Stats{}, err
	}

	if rowsAffected > 0 {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO post_stats (slug, views, likes) VALUES (?, 1, 0) 
            ON CONFLICT(slug) DO UPDATE SET views = views + 1;
        `, slug)
		if err != nil {
			return Stats{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Stats{}, err
	}

	return r.GetStats(ctx, slug)
}
```

This code is saying:
- Attempt to add a new row into ip_views, if a row already exists, just do nothing.
- If a row was inserted into ip_views, we update the post_stats table with the new view.

We do something similar for likes.

## Drawbacks to This Approach 
This isn't a perfect method to fix this, but I decided it's the right solution for something as small-scale as my blog.

One huge drawback is that users who are accessing the internet from a mobile data center can all share the same IP.
This means I won't get a new view from two unique users who happen to be using the same mobile tower. Alternative solutions
were high enough in complexity that this tradeoff was worth it to me.

## A Huge Gotcha: IPv6
To understand why IPv6 addresses pose a unique drawback and gotcha for this implementation, we need to understand how IPv6 addresses are composed.

IPv6 addresses consist of a "base" IP address (often shared by one entire household on the same router), and a "device interface". The device interface
is a unique part of the IPv6 which is used to uniquely identify each individual device running against a router. This poses a problem for my fix because
users who have IPv6 addresses can use the subnet ("device interface") to appear as any number of a very large amount of "unique" addresses, which defeats my deduplication logic.

The fix, then, is to mask the IPv6 address down to its "base" component. The tradeoff, of course, is now we're effectively only getting one view per household.
This tradeoff was worth it to me as opposed to a more robust solution, which would require significantly more effort to implement. 

## The Robust Solution
A robust solution would use a tool like Fingerprint to obtain a unique, anonymized device fingerprint per client and dedupe that way. I decided the effort
of doing this outweighs the benefits for my simple blog.

## Conclusion
I've reset the views and the likes on the blog. With the deduplication, the individual values of likes and views is now a lot more meaningful,
so starting from 0 made sense to me.

Don't trust the internet to not be douche bags. Lesson learned!
