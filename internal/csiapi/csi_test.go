package csiapi

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

// type mockControllerServer struct {
// 	csi.UnimplementedControllerServer
// }

// // func (m *mockControllerServer) CreateVolume(context.Context, *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
// // 	return &csi.CreateVolumeResponse{}, nil
// // }

// type mockNodeServer struct {
// 	csi.UnimplementedNodeServer
// }

// // func (m *mockNodeServer) NodePublishVolume(context.Context, *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
// // 	return &csi.NodePublishVolumeResponse{}, nil
// // }

// func TestNewCSIClient(t *testing.T) {
// 	clientOpts := []grpc.DialOption{
// 		grpc.WithContextDialer(bufDialer),
// 		grpc.WithTransportCredentials(insecure.NewCredentials()), // Use insecure credentials for testing
// 	}

// 	// Set up the mock server on bufconn
// 	t.Run("successful connection", func(t *testing.T) {
// 		// Create a mock server to handle the connection
// 		s := grpc.NewServer()
// 		csi.RegisterControllerServer(s, &mockControllerServer{})
// 		csi.RegisterNodeServer(s, &mockNodeServer{})

// 		// Start serving on the bufconn listener
// 		go func() {
// 			if err := s.Serve(lis); err != nil {
// 				t.Fatalf("failed to serve: %v", err)
// 			}
// 		}()

// 		// Attempt to connect to the bufnet (which will succeed)
// 		_, err := NewCSIClient("bufnet", clientOpts...)
// 		assert.NoError(t, err)
// 		// assert.NotNil(t, csiClient.DriverConn) // Access DriverConn directly from the actual CSIClient struct
// 	})

// 	t.Run("failed connection", func(t *testing.T) {
// 		// Modify the retry timeout for the test to allow quick failure
// 		CSIClientDialRetry = 1 * time.Second

// 		// Create a custom dialer that simulates failure for the "invalid" address
// 		customDialer := func(ctx context.Context, address string) (net.Conn, error) {
// 			if address == "invalid" {
// 				return nil, fmt.Errorf("failed to connect to invalid address")
// 			}
// 			return bufDialer(ctx, address) // Fall back to the original dialer for other addresses
// 		}

// 		// Use the custom dialer for this test case
// 		clientOptsWithCustomDialer := append(clientOpts, grpc.WithContextDialer(customDialer))

// 		// Attempt to connect to the invalid target
// 		_, err := NewCSIClient("invalid", clientOptsWithCustomDialer...)

// 		// Check for expected error
// 		assert.Error(t, err)
// 		assert.Contains(t, err.Error(), "failed to connect to invalid address")

// 		// // Ensure the DriverConn is nil if the connection failed
// 		// assert.Nil(t, csiClient.DriverConn)
// 	})
// }

func TestClient_Connected(t *testing.T) {
	client := &Client{}
	assert.False(t, client.Connected())

	client.DriverConn, _ = grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	assert.True(t, client.Connected())
}

func TestClient_Close(t *testing.T) {
	client := &Client{}
	assert.NoError(t, client.Close())

	client.DriverConn, _ = grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	assert.NoError(t, client.Close())
}

// func TestClient_ControllerUnpublishVolume(t *testing.T) {
// 	client := &Client{}
// 	ctx := context.Background()
// 	req := &csi.ControllerUnpublishVolumeRequest{}

// 	_, err := client.ControllerUnpublishVolume(ctx, req)
// 	assert.Error(t, err)

// 	client.ControllerClient = &CSIMock{}

// 	_, err = client.ControllerUnpublishVolume(ctx, req)
// 	assert.NoError(t, err)
// }

// func TestClient_NodeUnpublishVolume(t *testing.T) {
// 	client := &Client{}
// 	ctx := context.Background()
// 	req := &csi.NodeUnpublishVolumeRequest{}

// 	_, err := client.NodeUnpublishVolume(ctx, req)
// 	assert.Error(t, err)

// 	//client.NodeClient = &CSIMock{}
// 	_, err = client.NodeUnpublishVolume(ctx, req)
// 	assert.NoError(t, err)
// }

// func TestClient_NodeUnstageVolume(t *testing.T) {
// 	client := &Client{}
// 	ctx := context.Background()
// 	req := &csi.NodeUnstageVolumeRequest{}

// 	_, err := client.NodeUnstageVolume(ctx, req)
// 	assert.Error(t, err)

// 	//client.NodeClient = &CSIMock{}
// 	_, err = client.NodeUnstageVolume(ctx, req)
// 	assert.NoError(t, err)
// }

// func TestClient_ValidateVolumeHostConnectivity(t *testing.T) {
// 	client := &Client{}
// 	ctx := context.Background()
// 	req := &csiext.ValidateVolumeHostConnectivityRequest{}

// 	_, err := client.ValidateVolumeHostConnectivity(ctx, req)
// 	assert.Error(t, err)

// 	//client.PodmonClient = &CSIMock{}
// 	_, err = client.ValidateVolumeHostConnectivity(ctx, req)
// 	assert.NoError(t, err)
// }
