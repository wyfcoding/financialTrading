package client

import (
	"context"
	"fmt"

	executionv1 "github.com/wyfcoding/financialtrading/go-api/execution/v1"
	"github.com/wyfcoding/financialtrading/internal/connectivity/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gRPCExecutionClient struct {
	client executionv1.ExecutionServiceClient
}

func NewExecutionClient(addr string) (domain.ExecutionClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to execution service: %w", err)
	}
	return &gRPCExecutionClient{
		client: executionv1.NewExecutionServiceClient(conn),
	}, nil
}

func (c *gRPCExecutionClient) SubmitOrder(ctx context.Context, cmd domain.FIXOrderCommand) (string, error) {
	resp, err := c.client.ExecuteOrder(ctx, &executionv1.ExecuteOrderRequest{
		OrderId:  cmd.ClOrdID,
		UserId:   cmd.UserID,
		Symbol:   cmd.Symbol,
		Side:     cmd.Side,
		Price:    cmd.Price.String(),
		Quantity: cmd.Quantity.String(),
	})
	if err != nil {
		return "", err
	}
	return resp.ExecutionId, nil
}
