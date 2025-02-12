package csiapi

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
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

// TestControllerUnpublishVolume to test the method ControllerUnpublishVolume
func TestControllerUnpublishVolume(t *testing.T) {
	originalControllerClient := CSIClient.ControllerClient

	// Mock ControllerClient with a stubbed ControllerUnpublishVolume method
	CSIClient.ControllerClient = &mockControllerClient{
		ControllerUnpublishVolumeFunc: func(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest, opts ...grpc.CallOption) (*csi.ControllerUnpublishVolumeResponse, error) {
			if req.VolumeId == "fail" {
				return nil, errors.New("failed to unpublish volume")
			}
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		},
	}

	defer func() { CSIClient.ControllerClient = originalControllerClient }()

	client := &Client{}

	t.Run("Successful unpublish volume", func(t *testing.T) {
		req := &csi.ControllerUnpublishVolumeRequest{
			VolumeId: "vol-001",
		}
		resp, err := client.ControllerUnpublishVolume(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Failed to unpublish volume", func(t *testing.T) {
		req := &csi.ControllerUnpublishVolumeRequest{
			VolumeId: "fail",
		}
		resp, err := client.ControllerUnpublishVolume(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "failed to unpublish volume", err.Error())
	})
}

// TestNodeUnpublishVolume to test the method NodeUnpublishVolume
func TestNodeUnpublishVolume(t *testing.T) {
	originalNodeClient := CSIClient.NodeClient

	// Mock NodeClient with a stubbed NodeUnpublishVolume method
	CSIClient.NodeClient = &mockNodeClient{
		NodeUnpublishVolumeFunc: func(ctx context.Context, req *csi.NodeUnpublishVolumeRequest, opts ...grpc.CallOption) (*csi.NodeUnpublishVolumeResponse, error) {
			if req.VolumeId == "fail" {
				return nil, errors.New("failed to unpublish node volume")
			}
			return &csi.NodeUnpublishVolumeResponse{}, nil
		},
	}

	defer func() { CSIClient.NodeClient = originalNodeClient }()

	client := &Client{}

	t.Run("Successful unpublish node volume", func(t *testing.T) {
		req := &csi.NodeUnpublishVolumeRequest{
			VolumeId: "vol-001",
		}
		resp, err := client.NodeUnpublishVolume(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Failed to unpublish node volume", func(t *testing.T) {
		req := &csi.NodeUnpublishVolumeRequest{
			VolumeId: "fail",
		}
		resp, err := client.NodeUnpublishVolume(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "failed to unpublish node volume", err.Error())
	})
}

// TestNodeUnstageVolume to test the method NodeUnstageVolume
func TestNodeUnstageVolume(t *testing.T) {
	originalNodeClient := CSIClient.NodeClient

	// Mock NodeClient with a stubbed NodeUnstageVolume method
	CSIClient.NodeClient = &mockNodeClient{
		NodeUnstageVolumeFunc: func(ctx context.Context, req *csi.NodeUnstageVolumeRequest, opts ...grpc.CallOption) (*csi.NodeUnstageVolumeResponse, error) {
			if req.VolumeId == "fail" {
				return nil, errors.New("failed to unstage volume")
			}
			return &csi.NodeUnstageVolumeResponse{}, nil
		},
	}

	defer func() { CSIClient.NodeClient = originalNodeClient }()

	client := &Client{}

	t.Run("Successful unstage volume", func(t *testing.T) {
		req := &csi.NodeUnstageVolumeRequest{
			VolumeId: "vol-001",
		}
		resp, err := client.NodeUnstageVolume(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Failed to unstage volume", func(t *testing.T) {
		req := &csi.NodeUnstageVolumeRequest{
			VolumeId: "fail",
		}
		resp, err := client.NodeUnstageVolume(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "failed to unstage volume", err.Error())
	})
}

// TestValidateVolumeHostConnectivity to test the method ValidateVolumeHostConnectivity
func TestValidateVolumeHostConnectivity(t *testing.T) {
	originalPodmonClient := CSIClient.PodmonClient

	// Mock PodmonClient with a stubbed ValidateVolumeHostConnectivity method
	CSIClient.PodmonClient = &mockPodmonClient{
		ValidateVolumeHostConnectivityFunc: func(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest, opts ...grpc.CallOption) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
			if req.NodeId == "fail" {
				return nil, errors.New("failed to validate volume host connectivity")
			}
			return &csiext.ValidateVolumeHostConnectivityResponse{}, nil
		},
	}

	defer func() { CSIClient.PodmonClient = originalPodmonClient }()

	client := &Client{}

	t.Run("Successful validate volume host connectivity", func(t *testing.T) {
		req := &csiext.ValidateVolumeHostConnectivityRequest{
			NodeId: "node-001",
		}
		resp, err := client.ValidateVolumeHostConnectivity(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Failed to validate volume host connectivity", func(t *testing.T) {
		req := &csiext.ValidateVolumeHostConnectivityRequest{
			NodeId: "fail",
		}
		resp, err := client.ValidateVolumeHostConnectivity(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "failed to validate volume host connectivity", err.Error())
	})
}

// mockControllerClient is a mock implementation of the CSI ControllerClient
type mockControllerClient struct {
	csi.ControllerClient
	ControllerUnpublishVolumeFunc func(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest, opts ...grpc.CallOption) (*csi.ControllerUnpublishVolumeResponse, error)
}

// mockNodeClient is a mock implementation of the CSI NodeClient
type mockNodeClient struct {
	csi.NodeClient
	NodeUnpublishVolumeFunc func(ctx context.Context, req *csi.NodeUnpublishVolumeRequest, opts ...grpc.CallOption) (*csi.NodeUnpublishVolumeResponse, error)
	NodeUnstageVolumeFunc   func(ctx context.Context, req *csi.NodeUnstageVolumeRequest, opts ...grpc.CallOption) (*csi.NodeUnstageVolumeResponse, error)
}

func (m *mockControllerClient) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest, opts ...grpc.CallOption) (*csi.ControllerUnpublishVolumeResponse, error) {
	return m.ControllerUnpublishVolumeFunc(ctx, req, opts...)
}

func (m *mockNodeClient) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest, opts ...grpc.CallOption) (*csi.NodeUnpublishVolumeResponse, error) {
	return m.NodeUnpublishVolumeFunc(ctx, req, opts...)
}

func (m *mockNodeClient) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest, opts ...grpc.CallOption) (*csi.NodeUnstageVolumeResponse, error) {
	return m.NodeUnstageVolumeFunc(ctx, req, opts...)
}

// mockPodmonClient is a mock implementation of the PodmonClient
type mockPodmonClient struct {
	csiext.PodmonClient
	ValidateVolumeHostConnectivityFunc func(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest, opts ...grpc.CallOption) (*csiext.ValidateVolumeHostConnectivityResponse, error)
}

func (m *mockPodmonClient) ValidateVolumeHostConnectivity(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest, opts ...grpc.CallOption) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
	return m.ValidateVolumeHostConnectivityFunc(ctx, req, opts...)
}

func TestNewCSIClient(t *testing.T) {
	// Backup and restore original CSIClientDialRetry after tests
	originalCSIClientDialRetry := CSIClientDialRetry
	defer func() { CSIClientDialRetry = originalCSIClientDialRetry }()

	// For testing purposes, set a lower retry interval
	CSIClientDialRetry = 100 * time.Millisecond

	tests := []struct {
		name           string
		dialFunc       func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
		expectNoErrors bool
		timeout        time.Duration
	}{
		{
			// Successful connection
			name: "successful connection",
			dialFunc: func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
				return &grpc.ClientConn{}, nil
			},
			expectNoErrors: true,
			timeout:        1 * time.Second,
		},
		{
			// Failing connection initially then success
			name: "failing connection initially then success",
			dialFunc: func() func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
				counter := 0
				return func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
					counter++
					if counter > 3 {
						return &grpc.ClientConn{}, nil
					}
					return nil, errors.New("failed to connect")
				}
			}(),
			expectNoErrors: true,
			timeout:        2 * time.Second,
		},
		{
			// Failing connection permanently
			name: "failing connection permanently",
			dialFunc: func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
				return nil, errors.New("failed to connect")
			},
			expectNoErrors: false,
			timeout:        5 * CSIClientDialRetry, // Timeout after several retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalGrpcDialContext := getGrpcDialContext
			defer func() { getGrpcDialContext = originalGrpcDialContext }()
			getGrpcDialContext = tt.dialFunc

			done := make(chan bool)
			var client CSIApi
			var err error

			go func() {
				client, err = NewCSIClient("test.sock", grpc.WithInsecure())
				done <- true
			}()

			select {
			case <-done:
				if tt.expectNoErrors {
					assert.NoError(t, err)
					assert.NotNil(t, client)
					assert.NotNil(t, client.(*Client).DriverConn)
					assert.NotNil(t, client.(*Client).PodmonClient)
					assert.NotNil(t, client.(*Client).ControllerClient)
					assert.NotNil(t, client.(*Client).NodeClient)
				} else {
					assert.Error(t, err)
					assert.Nil(t, client)
				}
			case <-time.After(tt.timeout):
				// If we expect no errors and hit the timeout, it's an issue
				if tt.expectNoErrors {
					t.Errorf("Test %s timed out unexpectedly", tt.name)
				} else {
					t.Logf("Test %s timed out as expected", tt.name)
				}
			}
		})
	}
}
