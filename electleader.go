package leadership

import (
	"context"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPort = ":80"
	Elect       = "ELECT"
	Start       = "START"
	Leader      = "LEADER"
	Coordinator = "COORDINATOR"
	NotExists   = "NE"
)

type Address struct {
	Host      string
	ExtPort   string
	RpcClient LeaderElectionClient
}

type MemberMap struct {
	sync.RWMutex
	idMap map[string]Address
}

type Member struct {
	id         string
	addr       Address
	members    MemberMap
	PingLeader *time.Ticker
	leaderID   string
	Decision   chan string
	Server     *Server
}

func (m *Member) compareMemberUUIDs() {
	var refId = m.id
	for k := range m.members.idMap {
		id := k
		if strings.Compare(id, refId) > 0 {
			refId = id
		}
	}
	m.leaderID = refId
}

func (m *Member) getUUID() string {
	u1, err := uuid.NewV1()
	if err != nil {
		log.Fatalf("Error while generating UUID. Error - %s", err.Error())
	}
	return u1.String()
}

func (m *Member) startServer() {
	m.Server.UniqueID = m.getUUID()
	m.Server.CurrStatus = Start
	port := os.Getenv("GRPC_PORT")

	if len(port) == 0 {
		port = DefaultPort
	}
	lis, err := net.Listen("tcp", port)

	if err != nil {
		log.Fatalf("Failed to listen: %s", err.Error())
	}

	s := grpc.NewServer()

	RegisterLeaderElectionServer(s, m.Server)

	reflection.Register(s)

	s.GetServiceInfo()

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %s", err.Error())
	}
}

func (m *Member) checkIfLeaderAlreadyExists() string {
	for k, v := range m.members.idMap {
		id := k
		rep := m.checkMemberStatus(&v)
		if rep != nil && rep.Message == Leader {
			return id
		}
	}
	return NotExists
}

func (m *Member) checkMemberStatus(addr *Address) *StatusReply {
	addArr := []string{addr.Host, addr.ExtPort}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	caller := strings.Join(addArr, ":")
	r, _ := addr.RpcClient.Status(ctx, &StatusRequest{Caller: caller})
	log.Debugf("%s responded with %v", caller, r)
	return r
}

func (m *Member) isLeaderAlive() {
	addr := m.members.idMap[m.leaderID]

	for {
		select {
		case <-m.PingLeader.C:
			status := m.checkMemberStatus(&addr)
			if status == nil || strings.Compare(status.UniqueId, m.leaderID) != 0 {
				time.Sleep(time.Second)
				status1 := m.checkMemberStatus(&addr)
				if status1 == nil || strings.Compare(status1.UniqueId, m.leaderID) != 0 {
					m.Decision <- Elect
					m.leaderID = NotExists
					return
				}
			} else {
				log.Infof("%v running as * Coordinator * and %v running as * LEADER *", m.addr, addr)
			}
		}
	}
}

func (m *Member) electLeader(memberDiscoveryFunc func() []*Address) {
	var leaderId = NotExists

	peers := memberDiscoveryFunc()

	m.discoverSelfAndMembers(peers)

	leaderId = m.checkIfLeaderAlreadyExists()

	if strings.Compare(leaderId, NotExists) != 0 {
		m.leaderID = leaderId
		go m.setMemberStatus(Coordinator)
		log.Infof("--------%s:%s elected as the Leader---------", m.members.idMap[m.leaderID].Host, m.members.idMap[m.leaderID].ExtPort)
	} else {
		m.compareMemberUUIDs()
		if strings.Compare(m.id, m.leaderID) == 0 {
			go m.setMemberStatus(Leader)
			log.Infof("--------%s:%s elected as the Leader---------", m.addr.Host, m.members.idMap[m.leaderID].ExtPort)
		} else {
			go m.setMemberStatus(Coordinator)
			log.Infof("--------%s:%s elected as the Leader---------", m.members.idMap[m.leaderID].Host, m.members.idMap[m.leaderID].ExtPort)
		}
	}
}

func (m *Member) setMemberStatus(status string) {
	if status == Leader {
		m.Server.CurrStatus = Leader
		m.Decision <- Leader
	} else if status == Coordinator {
		m.Server.CurrStatus = Coordinator
		m.Decision <- Coordinator
	}
}

func createRpcClientFromAddress(addr *Address) {
	host := addr.Host
	port := addr.ExtPort
	addrArr := []string{host, port}
	addr.RpcClient = CreateInSecureClient(strings.Join(addrArr, ":"))
}

func (m *Member) discoverSelfAndMembers(peers []*Address) {
	if len(m.members.idMap) > 0 {
		for k := range m.members.idMap {
			delete(m.members.idMap, k)
		}
	}
	for _, v := range peers {
		addr := v
		createRpcClientFromAddress(addr)
		status := m.checkMemberStatus(addr)
		if status != nil {
			if (strings.Compare(m.id, NotExists) == 0 || m.id == "") && status.UniqueId == m.Server.UniqueID {
				m.addr = *addr
				m.id = status.UniqueId
			} else {
				m.members.idMap[status.UniqueId] = *addr
			}
		} else {
			log.Fatalf("**** ERROR -> MEMBER %v NOT REACHABLE *******", addr)
		}

	}
}

func (m *Member) Run(leaderJob func(), memberDisoveryFunc func() []*Address) {
	m.members.idMap = make(map[string]Address)

	go m.startServer()

	m.electLeader(memberDisoveryFunc)

	for {
		select {
		case dec := <-m.Decision:
			log.Infof("-----------Running in %s odedp")
			if dec == Elect {
				m.electLeader(memberDisoveryFunc)
			} else if dec == Coordinator {
				go m.isLeaderAlive()
			} else if dec == Leader {
				leaderJob()
			}
		}
	}

}
