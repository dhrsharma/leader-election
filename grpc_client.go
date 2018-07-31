package leadership

import (
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func CreateInSecureClient(address string) LeaderElectionClient {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Could not connect: %v", err)
	}
	return NewLeaderElectionClient(conn)
}
