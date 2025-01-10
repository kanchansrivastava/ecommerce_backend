package handlers

import (
	pb "cart-service/gen/proto"
	"cart-service/internal/cart"
	"context"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cartItemService struct {
	cartConf                              cart.Conf
	pb.UnimplementedCartItemServiceServer // Use the correct unimplemented server
}

// Constructor for cartItemService
func NewCartItemServiceHandler(cartConf cart.Conf) *cartItemService {
	return &cartItemService{
		cartConf: cartConf,
	}
}

func (c cartItemService) GetCartDetails(ctx context.Context, request *pb.GetCartDetailsRequest) (*pb.GetCartDetailsResponse, error) {
	// Log the incoming request for debugging purposes
	fmt.Printf("Received GetCartDetailsRequest: %+v\n", request)

	// Call GetActiveCartItems to fetch the active cart details
	cartResponse, err := c.cartConf.GetActiveCartItems(ctx, request.GetUserId())
	if err != nil {
		// Handle the error and return an appropriate gRPC status
		return nil, status.Errorf(codes.Internal, "failed to get cart details: %v", err)
	}

	// Map the cart items from the response to the protobuf-defined CartItem type
	var cartItems []*pb.CartItems // Correct type for individual cart items
	for _, item := range cartResponse.Items {
		cartItems = append(cartItems, &pb.CartItems{ // Ensure individual item type is pb.CartItem
			ProductID: item.ProductID, // Correct field naming based on protobuf definition
			Quantity:  item.Quantity,
		})
	}

	// Return the response
	return &pb.GetCartDetailsResponse{
		CartItems: cartItems,
	}, nil
}
