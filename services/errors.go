package services

import (
	"runtime/debug"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog"
)

func TransportUnavailable(err error) bool {
	if e, ok := status.FromError(err); ok && e.Code() == codes.Unavailable {
		klog.Fatalf("GRPCError: %s\n", e.Message())
		stack := debug.Stack()
		klog.Fatal(string(stack))
		return true
	}

	return false
}

func HandleGRPCError(err error) {
	if TransportUnavailable(err) {
		klog.Fatal("Communication with cloud agent failed. Exiting Process.")
	}
}
