package execution

import (
	"context"
)

const (
	topicDisconnect                   = "disconnect"
	topicHello                        = "hello"
	topicStatus                       = "status"
	topicTransactions                 = "transactions"
	topicNewPooledTransactionHashes   = "new_pooled_transaction_hashes"
	topicNewPooledTransactionHashes68 = "new_pooled_transaction_hashes_68"
)

func (c *Client) publishDisconnect(ctx context.Context, reason *Disconnect) {
	c.broker.Emit(topicDisconnect, reason)
}

func (c *Client) publishHello(ctx context.Context, status *Hello) {
	c.broker.Emit(topicHello, status)
}

func (c *Client) publishStatus(ctx context.Context, status *Status) {
	c.broker.Emit(topicStatus, status)
}

func (c *Client) publishTransactions(ctx context.Context, transactions *Transactions) {
	c.broker.Emit(topicTransactions, transactions)
}

func (c *Client) publishNewPooledTransactionHashes(ctx context.Context, hashes *NewPooledTransactionHashes) {
	c.broker.Emit(topicNewPooledTransactionHashes, hashes)
}

func (c *Client) publishNewPooledTransactionHashes68(ctx context.Context, hashes *NewPooledTransactionHashes68) {
	c.broker.Emit(topicNewPooledTransactionHashes68, hashes)
}

func (c *Client) handleSubscriberError(err error, topic string) {
	if err != nil {
		c.log.WithError(err).WithField("topic", topic).Error("Subscriber error")
	}
}

func (c *Client) OnDisconnect(ctx context.Context, handler func(ctx context.Context, reason *Disconnect) error) {
	c.broker.On(topicDisconnect, func(reason *Disconnect) {
		c.handleSubscriberError(handler(ctx, reason), topicDisconnect)
	})
}

func (c *Client) OnHello(ctx context.Context, handler func(ctx context.Context, status *Hello) error) {
	c.broker.On(topicHello, func(status *Hello) {
		c.handleSubscriberError(handler(ctx, status), topicHello)
	})
}

func (c *Client) OnStatus(ctx context.Context, handler func(ctx context.Context, status *Status) error) {
	c.broker.On(topicStatus, func(status *Status) {
		c.handleSubscriberError(handler(ctx, status), topicStatus)
	})
}

func (c *Client) OnTransactions(ctx context.Context, handler func(ctx context.Context, transactions *Transactions) error) {
	c.broker.On(topicTransactions, func(transactions *Transactions) {
		c.handleSubscriberError(handler(ctx, transactions), topicTransactions)
	})
}

func (c *Client) OnNewPooledTransactionHashes(ctx context.Context, handler func(ctx context.Context, hashes *NewPooledTransactionHashes) error) {
	c.broker.On(topicNewPooledTransactionHashes, func(hashes *NewPooledTransactionHashes) {
		c.handleSubscriberError(handler(ctx, hashes), topicNewPooledTransactionHashes)
	})
}

func (c *Client) OnNewPooledTransactionHashes68(ctx context.Context, handler func(ctx context.Context, hashes *NewPooledTransactionHashes68) error) {
	c.broker.On(topicNewPooledTransactionHashes68, func(hashes *NewPooledTransactionHashes68) {
		c.handleSubscriberError(handler(ctx, hashes), topicNewPooledTransactionHashes68)
	})
}
